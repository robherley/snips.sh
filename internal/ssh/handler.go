package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/rs/zerolog/log"
)

type SessionHandler struct {
	Config *config.Config
	DB     *db.DB
}

func (h *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		userSesh := &UserSession{sesh}

		// user requesting to download a file
		if userSesh.IsFileRequest() {
			h.Request(userSesh)
			return
		}

		// user entering interactive session w/ tui
		if userSesh.IsPTY() {
			h.Interactive(userSesh)
			return
		}

		// otherwise, it's a file upload
		h.Upload(userSesh)
	}
}

func (h *SessionHandler) Interactive(sesh *UserSession) {
	wish.Println(sesh, "👋 Welcome to snips.sh!")
	wish.Println(sesh, "🪪 You are user:", sesh.UserID())
	wish.Println(sesh, "🔑 Using key with fingerprint:", sesh.PublicKeyFingerprint())
}

func (h *SessionHandler) Request(sesh *UserSession) {
	userID := sesh.UserID()
	fileID := sesh.RequestedFileID()

	file := db.File{}
	if err := h.DB.First(&file, "id = ?", fileID).Error; err != nil {
		log.Error().Err(err).Msg("unable to lookup file")
		wish.Fatalf(sesh, "❌ File not found: %s\n", fileID)
		return
	}

	if file.Private && file.UserID != userID {
		log.Warn().Msg("attempted to access private file")
		wish.Fatalf(sesh, "❌ File not found: %s\n", fileID)
		return
	}

	wish.Print(sesh, string(file.Content))
}

func (h *SessionHandler) Upload(sesh *UserSession) {
	log := GetSessionLogger(sesh)

	flags, err := ParseUploadFlags(sesh)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Warn().Err(err).Msg("invalid user specified flags")
		}
		return
	}

	content := make([]byte, 0)
	size := uint64(0)
	for {
		buf := make([]byte, UploadBufferSize)
		n, err := sesh.Read(buf)
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF {
			log.Err(err).Msg("unable to read")
			wish.Fatalf(sesh, "❌ Error reading file")
			return
		}

		size += uint64(n)
		content = append(content, buf[:n]...)

		if size > MaxUploadSize {
			wish.Fatalf(sesh, "❌ File too large, max size is %s B\n", MaxUploadSize)
			return
		}

		if isEOF {
			if size == 0 {
				wish.Fatalln(sesh, "⚠️ Skipping upload, file is empty!")
				return
			}

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

			log.Info().Fields(map[string]interface{}{
				"id":         file.ID,
				"user_id":    file.UserID,
				"size":       file.Size,
				"expires_at": file.ExpiresAt,
				"private":    file.Private,
				"extension":  file.Extension,
			}).Msg("file uploaded")

			wish.Println(sesh, "✅ File Uploaded Successfully!")
			wish.Println(sesh, "💳 ID:", file.ID)
			wish.Println(sesh, "🏋️  Size:", humanize.Bytes(uint64(file.Size)))
			if file.Private {
				wish.Println(sesh, "🔐 Private")
			}
			if file.Extension != nil {
				wish.Println(sesh, "📁 Extension:", file.Extension)
			}
			if file.ExpiresAt != nil {
				wish.Println(sesh, "⏰ Expires:", file.ExpiresAt.Format(time.UnixDate))
			}

			httpAddr := fmt.Sprintf("%s:%d%s%s", h.Config.Host.External, h.Config.HTTP.Port, "/f/", file.ID)
			sshCommand := fmt.Sprintf("ssh %s%s@%s", FileRequestPrefix, file.ID, h.Config.Host.External)
			if h.Config.SSH.Port != 22 {
				sshCommand += fmt.Sprintf(" -p %d", h.Config.SSH.Port)
			}

			wish.Println(sesh, "🌐 URL:", httpAddr)
			wish.Println(sesh, "📠 SSH Command:", sshCommand)

			// wish.Println(sesh, "\n📱 Scan this QR code to download the file:\n")
			// config := qrterminal.Config{
			// 	Level:     qrterminal.L,
			// 	Writer:    sesh.Stderr(),
			// 	BlackChar: qrterminal.BLACK,
			// 	WhiteChar: qrterminal.WHITE,
			// 	QuietZone: 1,
			// }
			// qrterminal.GenerateWithConfig(httpAddr, config)

			return
		}
	}
}
