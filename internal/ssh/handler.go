package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/rs/zerolog/log"
)

type SessionHandler struct {
	Config *config.Config
	DB     *db.DB
	Signer *signer.Signer
}

func (h *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	return func(sesh ssh.Session) {
		userSesh := &UserSession{sesh}

		// user requesting to download a file
		if userSesh.IsFileRequest() {
			h.FileRequest(userSesh)
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
	// TODO(robherley): tui
	wish.Println(sesh, "👋 Welcome to snips.sh!")
	wish.Println(sesh, "🪪 You are user:", sesh.UserID())
	wish.Println(sesh, "🔑 Using key with fingerprint:", sesh.PublicKeyFingerprint())
}

func (h *SessionHandler) FileRequest(sesh *UserSession) {
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

	args := sesh.Command()
	if len(args) == 0 {
		h.DownloadFile(sesh, &file)
		return
	}

	switch args[0] {
	case "rm":
		h.DeleteFile(sesh, &file)
	case "sign":
		h.SignFile(sesh, &file)
	default:
		wish.Fatalf(sesh, "❌ Unknown command: %s\n", args[0])
	}
}

func (h *SessionHandler) DeleteFile(sesh *UserSession, file *db.File) {
	// TODO(robherley): add prompt & -f flag

	// this is a soft delete
	if err := h.DB.Delete(file).Error; err != nil {
		log.Error().Err(err).Msg("unable to delete file")
		wish.Fatalf(sesh, "❌ Error deleting file: %s\n", file.ID)
		return
	}

	log.Info().Fields(map[string]interface{}{
		"file_id": file.ID,
		"user_id": file.UserID,
	}).Msg("file deleted")

	wish.Printf(sesh, "✅ Deleted file: %s\n", file.ID)
}

func (h *SessionHandler) SignFile(sesh *UserSession, file *db.File) {
	if !file.Private {
		wish.Fatalf(sesh, "❌ Can only sign private files: %s\n", file.ID)
		return
	}

	flags := SignFlags{}
	if err := flags.Parse(sesh.Stderr(), sesh.Command()); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Warn().Err(err).Msg("invalid user specified flags")
			flags.PrintDefaults()
		}
		return
	}

	expires := time.Now().Add(flags.TTL)

	// only signing the path + queries of the URL
	pathToSign := url.URL{
		Path: fmt.Sprintf("/f/%s", file.ID),
		RawQuery: url.Values{
			"exp": []string{strconv.FormatInt(expires.Unix(), 10)},
		}.Encode(),
	}

	signedFileURL := h.Signer.SignURL(pathToSign)
	signedFileURL.Scheme = h.Config.URL.External.Scheme
	signedFileURL.Host = h.Config.URL.External.Host

	wish.Printf(sesh, "⏰ Signed file expires: %s\n", expires.Format(time.RFC3339))
	wish.Printf(sesh, "🔗 %s\n", signedFileURL.String())
}

func (h *SessionHandler) DownloadFile(sesh *UserSession, file *db.File) {
	wish.Print(sesh, string(file.Content))
}

func (h *SessionHandler) Upload(sesh *UserSession) {
	log := logger.From(sesh.Context())

	flags := UploadFlags{}
	if err := flags.Parse(sesh.Stderr(), sesh.Command()); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Warn().Err(err).Msg("invalid user specified flags")
			flags.PrintDefaults()
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
			wish.Fatalf(sesh, "❌ File too large, max size is %s\n", humanize.Bytes(MaxUploadSize))
			return
		}

		if isEOF {
			if size == 0 {
				wish.Fatalln(sesh, "⚠️ Skipping upload, file is empty!")
				return
			}

			file := db.File{
				Private: flags.Private,
				Content: content,
				Size:    size,
				UserID:  sesh.UserID(),
				Type:    renderer.DetectFileType(content, flags.Extension),
			}

			if err := h.DB.Create(&file).Error; err != nil {
				log.Err(err).Msg("unable to create file")
				wish.Fatalf(sesh, "❌ Error creating file")
				return
			}

			log.Info().Fields(map[string]interface{}{
				"file_id":   file.ID,
				"user_id":   file.UserID,
				"size":      file.Size,
				"private":   file.Private,
				"file_type": file.Type,
			}).Msg("file uploaded")

			wish.Println(sesh, "✅ File Uploaded Successfully!")
			wish.Println(sesh, "💳 ID:", file.ID)
			wish.Println(sesh, "🏋️  Size:", humanize.Bytes(uint64(file.Size)))
			wish.Println(sesh, "📁 Type:", file.Type)
			if file.Private {
				wish.Println(sesh, "🔐 Private")
			}

			httpAddr := h.Config.URL.External
			httpAddr.Path = fmt.Sprintf("/f/%s", file.ID)

			sshCommand := fmt.Sprintf("ssh %s%s@%s", FileRequestPrefix, file.ID, h.Config.URL.External.Hostname())
			if h.Config.SSH.Port != 22 {
				sshCommand += fmt.Sprintf(" -p %d", h.Config.SSH.Port)
			}

			wish.Println(sesh, "🔗 URL:", httpAddr.String())
			wish.Println(sesh, "📠 SSH Command:", sshCommand)

			return
		}
	}
}
