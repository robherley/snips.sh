package ssh

import (
	"errors"
	"io"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/bites"
	"github.com/robherley/snips.sh/internal/db"
)

type SessionHandler struct {
	DB *db.DB
}

func (handler *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		userSesh := &UserSession{sesh}

		_, _, isPty := sesh.Pty()
		if isPty {
			handler.Interactive(userSesh)
		} else {
			handler.Upload(userSesh)
		}
	}
}

func (h *SessionHandler) Interactive(sesh *UserSession) {
	wish.Println(sesh, "👋 Welcome to snips.sh!")
	wish.Println(sesh, "🪪 You are user:", sesh.UserID().String())
	wish.Println(sesh, "🔑 Using key with fingerprint:", sesh.PublicKeyFingerprint())
}

func (h *SessionHandler) Upload(sesh *UserSession) {
	log := GetSessionLogger(sesh)

	content := make([]byte, 0)
	size := int64(0)
	for {
		buf := make([]byte, UploadBufferSize)
		n, err := sesh.Read(buf)
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF {
			log.Err(err).Msg("unable to read")
			wish.Fatalf(sesh, "❌ Error reading file")
			return
		}

		size += int64(n)
		content = append(content, buf[:n]...)

		if size > MaxUploadSize {
			wish.Fatalf(sesh, "❌ File too large, max size is %s\n", bites.ByteSize(MaxUploadSize))
			return
		}

		if isEOF {
			file := db.File{
				Content: content,
				Size:    size,
				UserID:  sesh.UserID(),
			}

			if err := h.DB.Create(&file).Error; err != nil {
				log.Err(err).Msg("unable to create file")
				wish.Fatalf(sesh, "❌ Error creating file")
				return
			}

			log.Info().Int64("size", size).Msg("file uploaded")
			wish.Printf(sesh, "✅ File uploaded successfully (size: %s)\n", bites.ByteSize(size))
			return
		}
	}
}
