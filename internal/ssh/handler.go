package ssh

import (
	"errors"
	"flag"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/dustin/go-humanize"
	"github.com/muesli/termenv"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type SessionHandler struct {
	Config *config.Config
	DB     db.DB
}

func (h *SessionHandler) HandleFunc(_ ssh.Handler) ssh.Handler {
	lipgloss.SetColorProfile(termenv.ANSI256)

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
	log := logger.From(sesh.Context())
	metrics.IncrCounter([]string{"ssh", "session", "interactive"}, 1)

	pty, winChan, _ := sesh.Pty()

	files, err := h.DB.FindFilesByUser(sesh.Context(), sesh.UserID())
	if err != nil {
		sesh.Error(err, "Failed to retrieve files", "There was an error retrieving your files. Please try again.")
		return
	}

	program := tea.NewProgram(
		tui.New(
			sesh.Context(),
			h.Config,
			pty.Window.Width,
			pty.Window.Height,
			sesh.UserID(),
			sesh.PublicKeyFingerprint(),
			h.DB,
			files,
		),
		tea.WithInput(sesh),
		tea.WithOutput(sesh),
		tea.WithAltScreen(),
	)
	if program == nil {
		sesh.Error(ErrNilProgram, "Failed to create program", "There was an error establishing a connection. Please try again.")
		return
	}
	defer program.Kill()

	timer := time.NewTimer(MaxSessionDuration)
	defer timer.Stop()

	go func() {
		for {
			select {
			case <-sesh.Context().Done():
				program.Quit()
				return
			case <-timer.C:
				log.Warn().Msg("max session duration reached")
				program.Quit()
				return
			case w := <-winChan:
				if program != nil {
					program.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
				}
			}
		}
	}()

	if _, err := program.Run(); err != nil {
		log.Error().Err(err).Msg("app exited with error")
	}
}

func (h *SessionHandler) FileRequest(sesh *UserSession) {
	userID := sesh.UserID()
	fileID := sesh.RequestedFileID()

	file, err := h.DB.FindFile(sesh.Context(), fileID)
	if err != nil {
		sesh.Error(err, "Unable to get file", "File not found: %s", fileID)
		return
	}

	if file == nil {
		sesh.Error(ErrFileNotFound, "Unable to get file", "File not found: %s", fileID)
		return
	}

	if file.Private && file.UserID != userID {
		sesh.Error(ErrPrivateFileAccess, "Unable to get file", "File not found: %s", fileID)
		return
	}

	args := sesh.Command()
	if len(args) == 0 {
		h.DownloadFile(sesh, file)
		return
	}

	if file.UserID != userID {
		sesh.Error(ErrOpOnNonOwnedFile, "Unable to perform operation on file", "You do not own %q, therefore you cannot perform an operation on it.", fileID)
		return
	}

	switch args[0] {
	case "rm":
		h.DeleteFile(sesh, file)
	case "sign":
		h.SignFile(sesh, file)
	default:
		sesh.Error(ErrUnknownCommand, "Unknown command", "Unknown command specified: %q", args[0])
	}
}

func (h *SessionHandler) DeleteFile(sesh *UserSession, file *snips.File) {
	log := logger.From(sesh.Context())

	flags := DeleteFlags{}
	args := sesh.Command()[1:]
	if err := flags.Parse(sesh.Stderr(), args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Warn().Err(err).Msg("invalid user specified flags")
		}
		return
	}

	if !flags.Force {
		confirm := Confirm{}
		confirm.Questionf("Are you sure you want to delete %q?", file.ID)

		confirmed, err := confirm.Prompt(sesh)
		if err != nil {
			sesh.Error(err, "Unable to delete file", "There was an error deleting file: %q", file.ID)
			return
		}

		if !confirmed {
			noti := Notification{
				Title: "File Not Deleted ℹ️",
				Color: styles.Colors.Yellow,
				WithStyle: func(s *lipgloss.Style) {
					s.MarginTop(1)
				},
			}

			noti.Messagef("User chose not to delete file: %q", file.ID)
			noti.Render(sesh)
			return
		}
	}

	if err := h.DB.DeleteFile(sesh.Context(), file.ID); err != nil {
		sesh.Error(err, "Unable to delete file", "There was an error deleting file: %q", file.ID)
		return
	}

	metrics.IncrCounter([]string{"file", "delete"}, 1)

	log.Info().Str("file_id", file.ID).Msg("file deleted")

	noti := Notification{
		Color: styles.Colors.Green,
		Title: "File Deleted 🗑️",
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}

	noti.Messagef("Deleted file: %q", file.ID)
	noti.Render(sesh)
}

func (h *SessionHandler) SignFile(sesh *UserSession, file *snips.File) {
	log := logger.From(sesh.Context())

	if !file.Private {
		sesh.Error(ErrSignPublicFile, "Unable to sign file", "Can only sign private files, %q is not private.", file.ID)
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

	signedFileURL, expires := file.GetSignedURL(h.Config, flags.TTL)
	log.Info().Str("file_id", file.ID).Time("expires_at", expires).Msg("private file signed")

	metrics.IncrCounter([]string{"file", "sign"}, 1)

	noti := Notification{
		Color: styles.Colors.Cyan,
		Title: "Signed URL Generated 📝",
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Messagef("Expires at: %s", styles.C(styles.Colors.Yellow, expires.Format(time.RFC3339)))
	noti.Render(sesh)

	url := lipgloss.NewStyle().
		Foreground(styles.Colors.Blue).
		Underline(true).
		Render(signedFileURL.String())

	noti = Notification{
		Title:   "URL 🔐",
		Message: url,
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Render(sesh)
}

func (h *SessionHandler) DownloadFile(sesh *UserSession, file *snips.File) {
	wish.Print(sesh, string(file.Content))
}

func (h *SessionHandler) Upload(sesh *UserSession) {
	log := logger.From(sesh.Context())

	flags := UploadFlags{}
	if err := flags.Parse(sesh.Stderr(), sesh.Command()); err != nil {
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
			sesh.Error(err, "Unable to read file", "There was an error reading the file: %q", err.Error())
			return
		}

		size += uint64(n)
		content = append(content, buf[:n]...)

		if size > h.Config.Limits.FileSize {
			sesh.Error(ErrFileTooLarge, "Unable to upload file", "File too large, max size is %s", humanize.Bytes(h.Config.Limits.FileSize))
			return
		}

		if isEOF {
			if size == 0 {
				noti := Notification{
					Color:   styles.Colors.Yellow,
					Title:   "Skipping upload ℹ️",
					Message: "File is empty!",
					WithStyle: func(s *lipgloss.Style) {
						s.MarginTop(1)
					},
				}
				noti.Render(sesh)
				return
			}

			file := snips.File{
				Private: flags.Private,
				Content: content,
				Size:    size,
				UserID:  sesh.UserID(),
				Type:    renderer.DetectFileType(content, flags.Extension, h.Config.EnableGuesser),
			}

			if err := h.DB.CreateFile(sesh.Context(), &file, h.Config.Limits.FilesPerUser); err != nil {
				sesh.Error(err, "Unable to create file", "There was an error creating the file: %s", err.Error())
				return
			}

			metrics.IncrCounterWithLabels([]string{"file", "create"}, 1, []metrics.Label{
				{Name: "private", Value: strconv.FormatBool(file.Private)},
				{Name: "type", Value: file.Type},
			})

			log.Info().Fields(map[string]interface{}{
				"file_id":   file.ID,
				"user_id":   file.UserID,
				"size":      file.Size,
				"private":   file.Private,
				"file_type": file.Type,
			}).Msg("file uploaded")

			visibility := styles.C(styles.Colors.White, "public")
			if file.Private {
				visibility = styles.C(styles.Colors.Red, "private")
			}

			attrs := make([]string, 0)
			kvp := map[string]string{
				"type":       styles.C(styles.Colors.White, file.Type),
				"size":       styles.C(styles.Colors.White, humanize.Bytes(file.Size)),
				"visibility": visibility,
			}
			for k, v := range kvp {
				key := styles.C(styles.Colors.Muted, k+": ")
				attrs = append(attrs, key+v)
			}
			sort.Strings(attrs)

			noti := Notification{
				Color: styles.Colors.Green,
				Title: "File Uploaded 📤",
				WithStyle: func(s *lipgloss.Style) {
					s.MarginTop(1)
				},
			}
			noti.Messagef("id: %s\n%s", styles.C(styles.Colors.White, file.ID), strings.Join(attrs, styles.C(styles.Colors.Muted, " • ")))
			noti.Render(sesh)

			noti = Notification{
				Title:   "SSH 📠",
				Message: styles.C(styles.Colors.Blue, h.Config.SSHCommandForFile(file.ID)),
				WithStyle: func(s *lipgloss.Style) {
					s.MarginTop(1)
				},
			}
			noti.Render(sesh)

			url := lipgloss.NewStyle().
				Foreground(styles.Colors.Blue).
				Underline(true).
				Render(h.Config.HTTPAddressForFile(file.ID))

			noti = Notification{
				Title:   "URL 🔗",
				Message: url,
			}

			if file.Private {
				noti.Message = "<none> (requires a signed URL)"
			}

			noti.Render(sesh)

			return
		}
	}
}
