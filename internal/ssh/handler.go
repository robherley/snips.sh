package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/tui"
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
	pty, winChan, _ := sesh.Pty()

	files := []db.File{}
	if err := h.DB.Where("user_id = ?", sesh.UserID()).Find(&files).Error; err != nil {
		sesh.Error(err, "Failed to get files", "There was an error getting your files. Please try again.")
		return
	}

	m := tui.Model{
		Term:        pty.Term,
		Width:       pty.Window.Width,
		Height:      pty.Window.Height,
		UserID:      sesh.UserID(),
		Fingerprint: sesh.PublicKeyFingerprint(),
		Time:        time.Now(),
		Files:       files,
	}

	// todo: what is alt screen?
	prog := tea.NewProgram(&m, tea.WithInput(sesh), tea.WithOutput(sesh), tea.WithAltScreen())
	if prog == nil {
		sesh.Error(ErrNilProgram, "Failed to create program", "There was an error connecting to snips. Please try again.")
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-sesh.Context().Done():
				if prog != nil {
					prog.Quit()
					return
				}
			case w := <-winChan:
				if prog != nil {
					prog.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
				}
			case <-ticker.C:
				if prog != nil {
					prog.Send(tui.TimeMsg(time.Now()))
				}
			}
		}
	}()

	defer func() {
		if prog != nil {
			prog.Kill()
		}
	}()

	if _, err := prog.Run(); err != nil {
		log.Error().Err(err).Msg("app exited with error")
	}
}

func (h *SessionHandler) FileRequest(sesh *UserSession) {
	userID := sesh.UserID()
	fileID := sesh.RequestedFileID()

	file := db.File{}
	if err := h.DB.First(&file, "id = ?", fileID).Error; err != nil {
		sesh.Error(err, "Unable to get file", "File not found: %s\n", fileID)
		return
	}

	if file.Private && file.UserID != userID {
		sesh.Error(ErrPrivateFileAccess, "Unable to get file", "File not found: %s\n", fileID)
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
		sesh.Error(ErrUnknownCommand, "Unknown command", "Unknown command specified: %s\n", args[0])
	}
}

func (h *SessionHandler) DeleteFile(sesh *UserSession, file *db.File) {
	// TODO(robherley): add prompt & -f flag

	// this is a soft delete
	if err := h.DB.Delete(file).Error; err != nil {
		sesh.Error(err, "Unable to delete file", "There was an error deleting file: %s\n", file.ID)
		return
	}

	log.Info().Fields(map[string]interface{}{
		"file_id": file.ID,
		"user_id": file.UserID,
	}).Msg("file deleted")

	tui.PrintHeader(sesh, tui.HeaderSuccess, "File Deleted")
	wish.Printf(sesh, "Deleted file: %s\n", file.ID)
}

func (h *SessionHandler) SignFile(sesh *UserSession, file *db.File) {
	if !file.Private {
		sesh.Error(ErrSignPublicFile, "Unable to sign file", "Can only sign private files, %s is not private\n", file.ID)
		return
	}

	flags := SignFlags{}
	args := sesh.Command()[1:]
	if err := flags.Parse(sesh.Stderr(), args); err != nil {
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
	signedFileURL.Scheme = h.Config.HTTP.External.Scheme
	signedFileURL.Host = h.Config.HTTP.External.Host

	tui.PrintHeader(sesh, tui.HeaderSuccess, "Private File Signed")
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
			sesh.Error(err, "Unable to read file", "There was an error reading the file: %s\n", err.Error())
			return
		}

		size += uint64(n)
		content = append(content, buf[:n]...)

		if size > MaxUploadSize {
			sesh.Error(ErrFileTooLarge, "Unable to upload file", "File too large, max size is %s\n", humanize.Bytes(MaxUploadSize))
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
				sesh.Error(err, "Unable to create file", "There was an error creating the file: %s\n", err.Error())
				return
			}

			log.Info().Fields(map[string]interface{}{
				"file_id":   file.ID,
				"user_id":   file.UserID,
				"size":      file.Size,
				"private":   file.Private,
				"file_type": file.Type,
			}).Msg("file uploaded")

			tui.PrintHeader(sesh, tui.HeaderSuccess, "File Uploaded")
			wish.Println(sesh, "💳 ID:", file.ID)
			wish.Println(sesh, "🏋️  Size:", humanize.Bytes(uint64(file.Size)))
			wish.Println(sesh, "📁 Type:", file.Type)
			if file.Private {
				wish.Println(sesh, "🔐 Private")
			}

			httpAddr := h.Config.HTTP.External
			httpAddr.Path = fmt.Sprintf("/f/%s", file.ID)

			sshCommand := fmt.Sprintf("ssh %s%s@%s", FileRequestPrefix, file.ID, h.Config.SSH.External.Hostname())
			if sshPort := h.Config.SSH.External.Port(); sshPort != "22" {
				sshCommand += fmt.Sprintf(" -p %s", sshPort)
			}

			wish.Println(sesh, "🔗 URL:", httpAddr.String())
			wish.Println(sesh, "📠 SSH Command:", sshCommand)

			return
		}
	}
}
