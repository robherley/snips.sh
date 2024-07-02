package ssh

import (
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*ssh.Server
}

func New(cfg *config.Config, db db.DB) (*Service, error) {
	sessionHandler := &SessionHandler{
		Config: cfg,
		DB:     db,
	}

	authorizedKeys, err := cfg.SSHAuthorizedKeys()
	if err != nil {
		return nil, err
	}

	sshServer, err := wish.NewServer(
		wish.WithAddress(cfg.SSH.Internal.Host),
		wish.WithHostKeyPath(cfg.SSH.HostKeyPath),
		wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool {
			return true
		}),
		wish.WithPasswordAuth(func(_ ssh.Context, _ string) bool {
			// accept pw auth so we can display a helpful message
			return true
		}),
		// note: middleware is evaluated in reverse order
		wish.WithMiddleware(
			sessionHandler.HandleFunc,
			AssignUser(db, cfg.HTTP.External),
			WithAuthorizedKeys(authorizedKeys),
			BlockIfNoPublicKey,
			WithLogger,
			WithRequestID,
			WithSessionMetrics,
		),
	)
	if err != nil {
		return nil, err
	}

	return &Service{sshServer}, nil
}
