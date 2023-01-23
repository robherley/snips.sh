package ssh

import (
	"errors"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

func AssignUser(database *db.DB) func(next ssh.Handler) ssh.Handler {
	return func(next ssh.Handler) ssh.Handler {
		return func(sesh ssh.Session) {
			fingerprint := gossh.FingerprintSHA256(sesh.PublicKey())
			sesh.Context().SetValue(FingerprintContextKey, fingerprint)

			pubkey := db.PublicKey{}
			user := db.User{}

			// try to find a public key
			err := database.Where("fingerprint = ?", fingerprint).First(&pubkey).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Err(err).Msg("unable to find publickey")
				wish.Fatalln(sesh, "❌ Unable to authenticate")
				return
			}

			// upsert and create user if not found
			if errors.Is(err, gorm.ErrRecordNotFound) {
				user = db.User{
					PublicKeys: []db.PublicKey{
						{
							Fingerprint: fingerprint,
							Type:        sesh.PublicKey().Type(),
						},
					},
				}

				if err := database.Create(&user).Error; err != nil {
					log.Err(err).Msg("unable to create user")
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}

				pubkey = user.PublicKeys[0]
			} else {
				// find user
				err := database.Where("id = ?", pubkey.UserID).First(&user).Error
				if err != nil {
					log.Err(err).Msg("unable to find user")
					wish.Fatalln(sesh, "❌ Unable to authenticate")
					return
				}
			}

			sesh.Context().SetValue(UserIDContextKey, user.ID)
			logger := GetSessionLogger(sesh).With().Str("user_id", user.ID).Logger()
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
		requestID := id.MustGenerate()
		sesh.Context().SetValue(RequestIDContextKey, requestID)

		reqLog := log.With().Str("svc", "ssh").Str("addr", addr).Str("request_id", requestID).Logger()
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
