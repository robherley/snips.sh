package ssh

import (
	"database/sql"
	"errors"
	"io"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/file"
	"github.com/rs/zerolog/log"
)

type SessionHandler struct {
	db *sql.DB
}

func (sh *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		// if len(sesh.Command()) != 0 {
		// 	wish.Errorln(sesh, "unknown command")
		// 	sesh.Exit(1)
		// }

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
}

func (sh *SessionHandler) upload(sesh ssh.Session) {
	total := int64(0)
	for {
		buf := make([]byte, UploadBufferSize)
		n, err := sesh.Read(buf)
		total += int64(n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				wish.Printf(sesh, "✅ File uploaded successfully (size: %s)\n", file.ByteSize(total))
				log.Info().Int64("size", total).Msg("file uploaded")
				// TODO(robherley): save file blob to database
			} else {
				log.Err(err).Msg("unable to read")
				wish.Println(sesh, "❌ Error reading file")
				sesh.Exit(1)
			}
			return
		}
		if total > MaxUploadSize {
			wish.Printf(sesh, "❌ File too large, max size is %s\n", file.ByteSize(MaxUploadSize))
			sesh.Exit(1)
			return
		}
	}
}
