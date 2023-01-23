package ssh

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/bites"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/rs/zerolog/log"
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
			flags, err := ParseUploadFlags(sesh)
			if err != nil {
				if !errors.Is(err, flag.ErrHelp) {
					log.Warn().Err(err).Msg("invalid user specified flags")
				}
				return
			}

			handler.Upload(userSesh, flags)
		}
	}
}

func (h *SessionHandler) Interactive(sesh *UserSession) {
	wish.Println(sesh, "👋 Welcome to snips.sh!")
	wish.Println(sesh, "🪪 You are user:", sesh.UserID())
	wish.Println(sesh, "🔑 Using key with fingerprint:", sesh.PublicKeyFingerprint())
}

func (h *SessionHandler) Upload(sesh *UserSession, flags *UploadFlags) {
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
				Private:   flags.Private,
				Content:   content,
				Size:      size,
				UserID:    sesh.UserID(),
				Extension: flags.Extension,
			}

			if flags.TTL != nil {
				expiresAt := time.Now().Add(*flags.TTL)
				file.ExpiresAt = &expiresAt
			}

			if err := h.DB.Create(&file).Error; err != nil {
				log.Err(err).Msg("unable to create file")
				wish.Fatalf(sesh, "❌ Error creating file")
				return
			}

			details := map[string]interface{}{
				"id":         file.ID,
				"user_id":    file.UserID,
				"size":       file.Size,
				"expires_at": file.ExpiresAt,
				"private":    file.Private,
				"extension":  file.Extension,
			}

			log.Info().Fields(details).Msg("file uploaded")
			// TODO(robherley): print something nicer
			jsonStr, _ := json.MarshalIndent(details, "", "  ")
			wish.Printf(sesh, "%s\n", jsonStr)
			return
		}
	}
}
