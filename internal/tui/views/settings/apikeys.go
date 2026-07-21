package settings

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/timeutil"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// apiKeysView is the api key management page: a list of the user's keys,
// with a two-step (name, optional expiry) creation flow.
type apiKeysView struct {
	deps

	list          []*snips.APIKey
	cursor        int
	naming        bool            // typing the name for a new key
	expiring      bool            // typing the optional expiry for a new key
	nameInput     textinput.Model // name input for a new key
	ttlInput      textinput.Model // expiry input for a new key
	pendingName   string          // name entered for the key being created
	newToken      string          // token of the key just created, shown once
	armedDeleteID string          // key id armed for deletion (press x twice)
	feedback      feedback.Feedback
}

func newAPIKeysView(d deps) apiKeysView {
	name := textinput.New()
	name.CharLimit = snips.NameMaxLength
	name.SetWidth(30)
	name.Prompt = styles.BC(styles.Colors.Yellow, "> ")
	name.Placeholder = "name"

	ttl := textinput.New()
	ttl.CharLimit = 32
	ttl.SetWidth(30)
	ttl.Prompt = styles.BC(styles.Colors.Yellow, "> ")
	ttl.Placeholder = "expiry, e.g. 30d (optional)"

	return apiKeysView{
		deps:      d,
		nameInput: name,
		ttlInput:  ttl,
	}
}

// enter loads the user's keys and resets the page state.
func (m apiKeysView) enter() (apiKeysView, error) {
	m.cursor = 0
	m.naming = false
	m.expiring = false
	m.newToken = ""
	m.armedDeleteID = ""
	m.feedback = feedback.Feedback{}

	if err := m.reload(&m); err != nil {
		return m, err
	}

	return m, nil
}

// reload refreshes the key list from the database.
func (m apiKeysView) reload(into *apiKeysView) error {
	keys, err := m.db.FindAPIKeysByUser(m.ctx, m.user.ID)
	if err != nil {
		return fmt.Errorf("failed to load api keys: %w", err)
	}

	into.list = keys
	if into.cursor >= len(keys) {
		into.cursor = max(0, len(keys)-1)
	}

	return nil
}

func (m apiKeysView) update(msg tea.KeyPressMsg) (apiKeysView, result) {
	if m.naming {
		return m.updateNaming(msg)
	}

	if m.expiring {
		return m.updateExpiring(msg)
	}

	// any key other than a second x disarms a pending deletion
	if msg.String() != "x" {
		m.armedDeleteID = ""
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.list)-1 {
			m.cursor++
		}
	case "n":
		if uint64(len(m.list)) >= m.cfg.Limits.APIKeysPerUser {
			m.feedback = feedback.Error(fmt.Sprintf("api key limit reached (%d)", m.cfg.Limits.APIKeysPerUser))
			return m, result{}
		}
		m.naming = true
		m.feedback = feedback.Feedback{}
		m.nameInput.Reset()
		return m, result{cmd: m.nameInput.Focus()}
	case "x":
		return m.deleteKey()
	case "esc":
		return m, result{back: true}
	case "q":
		// the view captures input on deeper pages, so quit needs handling here
		return m, result{quit: true}
	}
	return m, result{}
}

// updateNaming handles the first creation step: a valid name moves on to the
// optional expiry.
func (m apiKeysView) updateNaming(msg tea.KeyPressMsg) (apiKeysView, result) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			m.feedback = feedback.Error("a name is required")
			return m, result{}
		}
		name, err := snips.NormalizeName(name)
		if err != nil {
			m.feedback = feedback.Error(err.Error())
			return m, result{}
		}
		m.pendingName = name
		m.naming = false
		m.expiring = true
		m.feedback = feedback.Feedback{}
		m.ttlInput.Reset()
		return m, result{cmd: m.ttlInput.Focus()}
	case "esc":
		m.naming = false
		m.feedback = feedback.Feedback{}
		return m, result{}
	}

	// everything else is typed into the name input
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, result{cmd: cmd}
}

// updateExpiring handles the second creation step: the optional expiry.
func (m apiKeysView) updateExpiring(msg tea.KeyPressMsg) (apiKeysView, result) {
	switch msg.String() {
	case "enter":
		return m.create()
	case "esc":
		m.expiring = false
		m.feedback = feedback.Feedback{}
		return m, result{}
	}

	// everything else is typed into the expiry input
	var cmd tea.Cmd
	m.ttlInput, cmd = m.ttlInput.Update(msg)
	return m, result{cmd: cmd}
}

// create mints a key named in the previous step, showing the token once.
func (m apiKeysView) create() (apiKeysView, result) {
	var expiresAt *time.Time
	if raw := strings.TrimSpace(m.ttlInput.Value()); raw != "" {
		ttl, err := timeutil.ParseDuration(raw)
		if err != nil || ttl <= 0 {
			m.feedback = feedback.Error("invalid expiry: use a duration like 30d or 12h")
			return m, result{}
		}
		expires := time.Now().UTC().Add(ttl)
		expiresAt = &expires
	}

	token, hash, err := snips.NewAPIKeyToken()
	if err != nil {
		m.feedback = feedback.Error("failed to create api key: " + err.Error())
		return m, result{}
	}

	key := &snips.APIKey{
		Name:      m.pendingName,
		TokenHash: hash,
		UserID:    m.user.ID,
		ExpiresAt: expiresAt,
	}

	if err := m.db.CreateAPIKey(m.ctx, key, m.cfg.Limits.APIKeysPerUser); err != nil {
		if errors.Is(err, db.ErrAPIKeyLimit) {
			m.feedback = feedback.Error(fmt.Sprintf("api key limit reached (%d)", m.cfg.Limits.APIKeysPerUser))
		} else {
			m.feedback = feedback.Error("failed to create api key: " + err.Error())
		}
		return m, result{}
	}

	metrics.IncrCounter([]string{"apikey", "create"}, 1)
	logger.From(m.ctx).Info("api key created", "api_key_id", key.ID, "user_id", key.UserID)

	m.expiring = false
	m.newToken = token
	m.feedback = feedback.Success("created api key " + key.DisplayName())

	if err := m.reload(&m); err != nil {
		m.feedback = feedback.Error(err.Error())
	}
	return m, result{}
}

// deleteKey removes the selected key, requiring x to be pressed twice.
func (m apiKeysView) deleteKey() (apiKeysView, result) {
	if len(m.list) == 0 {
		return m, result{}
	}

	selected := m.list[m.cursor]
	if m.armedDeleteID != selected.ID {
		m.armedDeleteID = selected.ID
		m.feedback = feedback.Error("press x again to delete " + selected.DisplayName())
		return m, result{}
	}

	deleted, err := m.db.DeleteAPIKey(m.ctx, selected.ID, m.user.ID)
	if err != nil || !deleted {
		msg := "failed to delete api key"
		if err != nil {
			msg += ": " + err.Error()
		}
		m.feedback = feedback.Error(msg)
		return m, result{}
	}

	metrics.IncrCounter([]string{"apikey", "delete"}, 1)
	logger.From(m.ctx).Info("api key deleted", "api_key_id", selected.ID, "user_id", m.user.ID)

	m.armedDeleteID = ""
	m.newToken = ""
	m.feedback = feedback.Success("deleted api key " + selected.DisplayName())

	if err := m.reload(&m); err != nil {
		m.feedback = feedback.Error(err.Error())
	}
	return m, result{}
}

// rows renders the key list, the creation inputs, and the one-time token
// reveal after a mint.
func (m apiKeysView) rows() []string {
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)

	rows := []string{
		mutedStyle.Render(fmt.Sprintf("keys for the rest api (%d/%d)", len(m.list), m.cfg.Limits.APIKeysPerUser)),
		"",
	}

	if len(m.list) == 0 {
		rows = append(rows, mutedStyle.Render("no api keys yet — press n to create one"))
	}

	// pad the name column so the dates line up across rows
	nameWidth := 0
	for _, key := range m.list {
		nameWidth = max(nameWidth, lipgloss.Width(key.DisplayName()))
	}

	for i, key := range m.list {
		cursor := "  "
		nameStyle := mutedStyle
		if i == m.cursor && !m.naming && !m.expiring {
			cursor = styles.BC(m.accent(), "→ ")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		}

		lastUsed := "never used"
		if key.LastUsedAt != nil {
			lastUsed = "last used " + key.LastUsedAt.UTC().Format("2006-01-02")
		}

		row := cursor + nameStyle.Width(nameWidth).Render(key.DisplayName()) +
			mutedStyle.Render(fmt.Sprintf("  ·  created %s  ·  %s", key.CreatedAt.UTC().Format("2006-01-02"), lastUsed))
		switch {
		case key.IsExpired():
			row += mutedStyle.Render("  ·  ") + styles.C(styles.Colors.Red, "expired")
		case key.ExpiresAt != nil:
			row += mutedStyle.Render("  ·  expires " + key.ExpiresAt.UTC().Format("2006-01-02"))
		}
		if m.armedDeleteID == key.ID {
			row += "  " + styles.C(styles.Colors.Red, "(press x again)")
		}
		rows = append(rows, row)
	}

	if m.naming {
		rows = append(rows, "", m.nameInput.View())
	}

	if m.expiring {
		rows = append(rows, "", mutedStyle.Render("name: ")+m.pendingName, m.ttlInput.View())
	}

	if m.newToken != "" {
		rows = append(rows,
			"",
			styles.C(styles.Colors.Yellow, "copy your new api key now, it won't be shown again:"),
			styles.C(styles.Colors.White, m.newToken),
		)
	}

	if !m.feedback.Empty() {
		rows = append(rows, "", m.feedback.View())
	}

	return rows
}

func (m apiKeysView) keys() help.KeyMap {
	if m.naming || m.expiring {
		return apiKeyNamingKeys
	}
	return apiKeysKeys
}

// apiKeysKeyMap is shown while navigating the api keys page.
type apiKeysKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	New    key.Binding
	Delete key.Binding
	Esc    key.Binding
	Quit   key.Binding
}

func (k apiKeysKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.New, k.Delete, k.Esc, k.Quit}
}

func (k apiKeysKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.New, k.Delete, k.Esc, k.Quit},
	}
}

var apiKeysKeys = apiKeysKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new key"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete key"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// apiKeyNamingKeyMap is shown while a creation input is focused; every other
// key is typed into the input.
type apiKeyNamingKeyMap struct {
	Enter key.Binding
	Esc   key.Binding
	Quit  key.Binding
}

func (k apiKeyNamingKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Esc, k.Quit}
}

func (k apiKeyNamingKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Esc, k.Quit}}
}

var apiKeyNamingKeys = apiKeyNamingKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "next"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}
