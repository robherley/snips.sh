package ssh

import (
	"log/slog"
	"net/url"
	"time"

	"github.com/armon/go-metrics"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	gossh "golang.org/x/crypto/ssh"
)

// AssignUser will attempt to match a user with a public key fingerprint.
// If a user is not found, one will be created with the current fingerprint attached.
func AssignUser(database db.DB, externalAddress url.URL) func(next ssh.Handler) ssh.Handler {
	return func(next ssh.Handler) ssh.Handler {
		return func(sesh ssh.Session) {
			fingerprint := gossh.FingerprintSHA256(sesh.PublicKey())
			sesh.Context().SetValue(FingerprintContextKey, fingerprint)

			var (
				user   *snips.User
				pubkey *snips.PublicKey
				err    error
			)

			// try to find a public key
			pubkey, err = database.FindPublicKeyByFingerprint(sesh.Context(), fingerprint)
			if err != nil {
				slog.Error("unable to find publickey", "err", err)
				wish.Fatalln(sesh, "❌ Unable to authenticate")
				return
			}

			// upsert and create user if not found
			if pubkey == nil {
				// welcome message over stderr so it doesn't jank up the output if they're piping
				wish.Errorln(sesh, "Welcome to snips.sh! 👋")
				wish.Errorln(sesh, "Please take a moment to read our terms of service.")
				wish.Errorf(sesh, "🔗 %s/docs/terms-of-service.md\n", externalAddress.String())

				pubkey = &snips.PublicKey{
					Fingerprint: fingerprint,
					Type:        sesh.PublicKey().Type(),
				}

				user, err = database.CreateUserWithPublicKey(sesh.Context(), pubkey)
				if err != nil {
					slog.Error("unable to create user", "err", err)
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
				metrics.IncrCounter([]string{"user", "create"}, 1)
			} else {
				// find user
				user, err = database.FindUser(sesh.Context(), pubkey.UserID)
				if err != nil || user == nil {
					slog.Error("unable to find user", "err", err)
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
			}

			sesh.Context().SetValue(UserIDContextKey, user.ID)

			log := logger.From(sesh.Context()).With("user_id", user.ID)
			sesh.Context().SetValue(logger.ContextKey, log)

			log.Info("user authenticated")
			metrics.IncrCounter([]string{"ssh", "session", "authenticated"}, 1)

			next(sesh)
		}
	}
}

// BlockIfNoPublicKey will stop any SSH connections that aren't using public key authentication.
// If blocked, it will print a helpful message to the user.
func BlockIfNoPublicKey(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		if key := sesh.PublicKey(); key == nil {
			metrics.IncrCounter([]string{"ssh", "session", "no_public_key"}, 1)
			wish.Println(sesh, "❌ Unfortunately snips.sh only supports public key authentication.")
			wish.Println(sesh, "🔐 Please generate a keypair and try again.")
			_ = sesh.Exit(1)
			return
		}
		next(sesh)
	}
}

// WithRequestID will generate a unique request ID for each SSH session.
func WithRequestID(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		requestID := id.New()
		sesh.Context().SetValue(RequestIDContextKey, requestID)
		next(sesh)
	}
}

// WithLogger will create a logger for each SSH session.
func WithLogger(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		addr := sesh.RemoteAddr().String()
		start := time.Now()
		requestID := sesh.Context().Value(RequestIDContextKey).(string)

		reqLog := slog.With("svc", "ssh", "addr", addr, "request_id", requestID)
		sesh.Context().SetValue(logger.ContextKey, reqLog)

		reqLog.Info("connected")
		next(sesh)
		reqLog.Info("disconnected", "dur", time.Since(start))
	}
}

// WithSessionMetrics will record metrics for each SSH session.
func WithSessionMetrics(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		start := time.Now()
		metrics.IncrCounter([]string{"ssh", "session", "connected"}, 1)
		next(sesh)
		metrics.MeasureSince([]string{"ssh", "session", "duration"}, start)
	}
}

// WithAuthorizedKeys will block any SSH connections that aren't using a public key in the authorized key list.
// If authorizedKeys is empty, this middleware will be a no-op.
func WithAuthorizedKeys(authorizedKeys []ssh.PublicKey) func(next ssh.Handler) ssh.Handler {
	if len(authorizedKeys) == 0 {
		return func(next ssh.Handler) ssh.Handler {
			return next
		}
	}

	slog.Debug("using SSH allowlist", "allowed_keys", len(authorizedKeys))

	return func(next ssh.Handler) ssh.Handler {
		return func(sesh ssh.Session) {
			for _, key := range authorizedKeys {
				if ssh.KeysEqual(key, sesh.PublicKey()) {
					next(sesh)
					return
				}
			}

			metrics.IncrCounter([]string{"ssh", "session", "not_authorized_key"}, 1)
			wish.Println(sesh, "❌ Public key not authorized.")
			_ = sesh.Exit(1)
		}
	}
}
