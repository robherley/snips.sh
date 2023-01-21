package ssh

import (
	"errors"
	"io"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/file"
)

type SessionHandler struct {
	db *db.DB
}

func (sh *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		_, _, isPty := sesh.Pty()
		if isPty {
			sh.interactive(sesh)
		} else {
			sh.upload(sesh)
		}
	}
}

func (sh *SessionHandler) interactive(sesh ssh.Session) {
	wish.Println(sesh, "👋 Welcome to snips.sh!")
	wish.Println(sesh, "🪪 You are user:", sesh.Context().Value(UserIDContextKey))
	wish.Println(sesh, "🔑 Using key with fingerprint:", sesh.Context().Value(FingerprintContextKey))
}

func (sh *SessionHandler) upload(sesh ssh.Session) {
	log := GetSessionLogger(sesh)

	total := int64(0)
	for {
		buf := make([]byte, UploadBufferSize)
		n, err := sesh.Read(buf)
		total += int64(n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Info().Int64("size", total).Msg("file uploaded")
				wish.Printf(sesh, "✅ File uploaded successfully (size: %s)\n", file.ByteSize(total))
				// TODO(robherley): save file blob to database
			} else {
				log.Err(err).Msg("unable to read")
				wish.Fatalf(sesh, "❌ Error reading file")
			}
			return
		}
		if total > MaxUploadSize {
			wish.Fatalf(sesh, "❌ File too large, max size is %s\n", file.ByteSize(MaxUploadSize))
			return
		}
	}
}
