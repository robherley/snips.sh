package ssh

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// APIKey dispatches the `api-key <create|ls|rm>` command for managing REST
// API keys.
func (h *SessionHandler) APIKey(sesh *UserSession) {
	args := sesh.Command()[1:]
	if len(args) == 0 {
		sesh.Error(ErrUnknownCommand, "Unknown command", "Usage: %s <create|ls|rm>", APIKeyCommand)
		return
	}

	switch args[0] {
	case "create":
		h.CreateAPIKey(sesh, args[1:])
	case "ls":
		h.ListAPIKeys(sesh)
	case "rm":
		h.RemoveAPIKey(sesh, args[1:])
	default:
		sesh.Error(ErrUnknownCommand, "Unknown command", "Unknown subcommand %q, expected <create|ls|rm>", args[0])
	}
}

func (h *SessionHandler) CreateAPIKey(sesh *UserSession, args []string) {
	log := logger.From(sesh.Context())

	flags := APIKeyCreateFlags{}
	if err := flags.Parse(sesh.Stderr(), args); err != nil {
		switch {
		case errors.Is(err, flag.ErrHelp):
		case errors.Is(err, ErrFlagRequired):
			sesh.Error(err, "Unable to create api key", "A name is required, e.g.: api-key create -name ci")
		default:
			log.Warn("invalid user specified flags", "err", err)
			sesh.Error(err, "Error parsing flag", "Error: %q", err.Error())
		}
		return
	}

	name, err := snips.NormalizeName(flags.Name)
	if err != nil {
		sesh.Error(err, "Unable to create api key", "Invalid name %q: %s", flags.Name, err.Error())
		return
	}

	token, hash, err := snips.NewAPIKeyToken()
	if err != nil {
		sesh.Error(err, "Unable to create api key", "There was an error creating the api key. Please try again.")
		return
	}

	key := &snips.APIKey{
		Name:      name,
		TokenHash: hash,
		UserID:    sesh.UserID(),
	}
	if flags.TTL > 0 {
		expires := time.Now().UTC().Add(flags.TTL)
		key.ExpiresAt = &expires
	}

	if err := h.DB.CreateAPIKey(sesh.Context(), key, h.Config.Limits.APIKeysPerUser); err != nil {
		if errors.Is(err, db.ErrAPIKeyLimit) {
			sesh.Error(err, "Unable to create api key", "You already have the maximum of %d api keys. Remove one with: api-key rm <id>", h.Config.Limits.APIKeysPerUser)
			return
		}
		sesh.Error(err, "Unable to create api key", "There was an error creating the api key. Please try again.")
		return
	}

	metrics.IncrCounter([]string{"apikey", "create"}, 1)
	log.Info("api key created", "api_key_id", key.ID, "user_id", key.UserID)

	noti := Notification{
		Color: styles.Colors.Green,
		Title: "API Key Created 🔑",
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	created := styles.C(styles.Colors.White, key.DisplayName())
	if key.ExpiresAt != nil {
		created += "\nexpires: " + styles.C(styles.Colors.Yellow, key.ExpiresAt.Format(time.RFC3339))
	}
	noti.Messagef("%s", created)
	noti.Render(sesh)

	noti = Notification{
		Color:   styles.Colors.Blue,
		Title:   "Token 🔐",
		Message: styles.C(styles.Colors.Yellow, token),
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Render(sesh)

	noti = Notification{
		Color:   styles.Colors.Red,
		Title:   "Heads Up ⚠️",
		Message: "Save this token now, it will not be shown again.\nUse it with: " + styles.C(styles.Colors.Blue, fmt.Sprintf("curl -H %q %s/api/v1/user", "Authorization: Bearer <token>", strings.TrimSuffix(h.Config.HTTP.External.String(), "/"))),
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Render(sesh)
}

func (h *SessionHandler) ListAPIKeys(sesh *UserSession) {
	keys, err := h.DB.FindAPIKeysByUser(sesh.Context(), sesh.UserID())
	if err != nil {
		sesh.Error(err, "Unable to list api keys", "There was an error listing your api keys. Please try again.")
		return
	}

	if len(keys) == 0 {
		noti := Notification{
			Color:   styles.Colors.Yellow,
			Title:   "No API Keys ℹ️",
			Message: "Create one with: api-key create -name <name>",
			WithStyle: func(s *lipgloss.Style) {
				s.MarginTop(1)
			},
		}
		noti.Render(sesh)
		return
	}

	var table strings.Builder
	tabs := tabwriter.NewWriter(&table, 1, 0, 2, ' ', 0)
	fmt.Fprintln(tabs, "KEY\tCREATED\tLAST USED\tEXPIRES")
	for _, key := range keys {
		lastUsed := "never"
		if key.LastUsedAt != nil {
			lastUsed = key.LastUsedAt.UTC().Format(time.RFC3339)
		}
		expires := "never"
		switch {
		case key.IsExpired():
			expires = "expired"
		case key.ExpiresAt != nil:
			expires = key.ExpiresAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(tabs, "%s\t%s\t%s\t%s\n", key.DisplayName(), key.CreatedAt.UTC().Format(time.RFC3339), lastUsed, expires)
	}
	if err := tabs.Flush(); err != nil {
		sesh.Error(err, "Unable to list api keys", "There was an error listing your api keys. Please try again.")
		return
	}

	_, _ = fmt.Fprint(sesh, table.String())
}

func (h *SessionHandler) RemoveAPIKey(sesh *UserSession, args []string) {
	log := logger.From(sesh.Context())

	if len(args) == 0 || args[0] == "" {
		sesh.Error(ErrAPIKeyIDRequired, "Unable to remove api key", "Provide an api key, e.g.: api-key rm <name> (list keys with: api-key ls)")
		return
	}

	// keys are referenced by name when they have one, so accept either
	keys, err := h.DB.FindAPIKeysByUser(sesh.Context(), sesh.UserID())
	if err != nil {
		sesh.Error(err, "Unable to remove api key", "There was an error removing api key: %q", args[0])
		return
	}

	var target *snips.APIKey
	for _, key := range keys {
		if key.ID == args[0] || (key.Name != "" && strings.EqualFold(key.Name, args[0])) {
			target = key
			break
		}
	}

	if target == nil {
		sesh.Error(ErrAPIKeyNotFound, "Unable to remove api key", "API key not found: %q", args[0])
		return
	}

	deleted, err := h.DB.DeleteAPIKey(sesh.Context(), target.ID, sesh.UserID())
	if err != nil || !deleted {
		sesh.Error(err, "Unable to remove api key", "There was an error removing api key: %q", args[0])
		return
	}

	metrics.IncrCounter([]string{"apikey", "delete"}, 1)
	log.Info("api key deleted", "api_key_id", target.ID, "user_id", sesh.UserID())

	noti := Notification{
		Color: styles.Colors.Green,
		Title: "API Key Removed 🗑️",
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Messagef("Removed api key: %q", target.DisplayName())
	noti.Render(sesh)
}
