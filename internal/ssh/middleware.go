package ssh

import (
	"database/sql"
	"errors"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	gossh "golang.org/x/crypto/ssh"
)

func CreateAndAttachUser(database *db.DB) func(next ssh.Handler) ssh.Handler {
	return func(next ssh.Handler) ssh.Handler {
		return func(sesh ssh.Session) {
			fingerprint := gossh.FingerprintSHA256(sesh.PublicKey())
			sesh.Context().SetValue(FingerprintContextKey, fingerprint)

			var (
				userID string
				err    error
			)

			userID, err = database.FindUserIDByFingerprint(sesh.Context(), fingerprint)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					userID, err = database.CreateNewUser(sesh.Context(), fingerprint, sesh.PublicKey().Type())
					if err != nil {
						log.Err(err).Msg("unable to create user")
						wish.Fatalln(sesh, "❌ Unable to authenticate")
						return
					}
				} else {
					log.Err(err).Msg("unable to find user")
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
			}

			sesh.Context().SetValue(UserIDContextKey, userID)

			logger := GetSessionLogger(sesh).With().Str("user_id", userID).Logger()
			SetSessionLogger(sesh, &logger)
			logger.Info().Msg("user authenticated")

			next(sesh)
		}
	}
}

func BlockIfNoPublicKey(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		if key := sesh.PublicKey(); key == nil {
			wish.Println(sesh, "❌ Unfortunately snips.sh only supports public key authentication.")
			wish.Println(sesh, "🔐 Please generate a keypair and try again.")
			sesh.Exit(1)
			return
		}
		next(sesh)
	}
}

func WithLogger(next ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		addr := sesh.RemoteAddr().String()
		start := time.Now()
		requestID := ksuid.New().String()
		sesh.Context().SetValue(RequestIDContextKey, requestID)

		reqLog := log.With().Str("addr", addr).Str("request_id", requestID).Logger()
		SetSessionLogger(sesh, &reqLog)

		reqLog.Info().Msg("connected")
		next(sesh)
		reqLog.Info().Dur("dur", time.Since(start)).Msg("disconnected")
	}
}

func SetSessionLogger(sesh ssh.Session, logger *zerolog.Logger) {
	sesh.Context().SetValue(LoggerContextKey, logger)
}

func GetSessionLogger(sesh ssh.Session) *zerolog.Logger {
	return sesh.Context().Value(LoggerContextKey).(*zerolog.Logger)
}
