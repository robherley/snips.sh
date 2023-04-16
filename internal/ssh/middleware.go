package ssh

import (
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
)

// AssignUser will attempt to match a user with a public key fingerprint.
// If a user is not found, one will be created with the current fingerprint attached.
func AssignUser(database db.DB) func(next ssh.Handler) ssh.Handler {
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
				log.Err(err).Msg("unable to find publickey")
				wish.Fatalln(sesh, "❌ Unable to authenticate")
				return
			}

			// upsert and create user if not found
			if pubkey == nil {
				pubkey = &snips.PublicKey{
					Fingerprint: fingerprint,
					Type:        sesh.PublicKey().Type(),
				}

				user, err = database.CreateUserWithPublicKey(sesh.Context(), pubkey)
				if err != nil {
					log.Err(err).Msg("unable to create user")
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
			} else {
				// find user
				user, err = database.FindUser(sesh.Context(), pubkey.UserID)
				if err != nil || user == nil {
					log.Err(err).Msg("unable to find user")
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
			}

			sesh.Context().SetValue(UserIDContextKey, user.ID)

			logger.From(sesh.Context()).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("user_id", user.ID)
			})
			logger.From(sesh.Context()).Info().Msg("user authenticated")

			next(sesh)
		}
	}
}

// BlockIfNoPublicKey will stop any SSH connections that aren't using public key authentication.
// If blocked, it will print a helpful message to the user.
func BlockIfNoPublicKey(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		if key := sesh.PublicKey(); key == nil {
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

		reqLog := log.With().Str("svc", "ssh").Str("addr", addr).Str("request_id", requestID).Logger()
		sesh.Context().SetValue(logger.ContextKey, &reqLog)

		reqLog.Info().Msg("connected")
		next(sesh)
		reqLog.Info().Dur("dur", time.Since(start)).Msg("disconnected")
	}
}
