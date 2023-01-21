package ssh

import (
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/rs/zerolog/log"
)

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
		isPubKey := sesh.PublicKey() != nil
		_, _, isPty := sesh.Pty()
		start := time.Now()

		reqLog := log.With().Str("addr", addr).Bool("is_pubkey", isPubKey).Bool("is_pty", isPty).Logger()

		reqLog.Info().Msg("connected")
		next(sesh)
		reqLog.Info().Dur("dur", time.Since(start)).Msg("disconnected")
	}
}
