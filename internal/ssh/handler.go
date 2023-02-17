package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

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
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/tui"
	"github.com/robherley/snips.sh/internal/tui/components/filelist"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/rs/zerolog/log"
)

type SessionHandler struct {
	Config *config.Config
	DB     *db.DB
	Signer *signer.Signer
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
	pty, winChan, _ := sesh.Pty()

	files := []filelist.ListItem{}
	if err := h.DB.Model(&db.File{}).Where("user_id = ?", sesh.UserID()).Order("created_at DESC").Find(&files).Error; err != nil {
		sesh.Error(err, "Unable to get files", "There was an error requesting files. Please try again.")
		return
	}

	newTUI := tui.New(
		pty.Window,
		sesh.UserID(),
		sesh.PublicKeyFingerprint(),
		h.DB,
		files,
	)

	prog := tea.NewProgram(newTUI, tea.WithInput(sesh), tea.WithOutput(sesh), tea.WithAltScreen())
	if prog == nil {
		sesh.Error(ErrNilProgram, "Failed to create program", "There was an error establishing a connection. Please try again.")
		return
	}

	timer := time.NewTimer(MaxSessionDuration)
	defer timer.Stop()

	go func() {
		for {
			select {
			case <-sesh.Context().Done():
				prog.Quit()
				return
			case <-timer.C:
				prog.Quit()
				return
			case w := <-winChan:
				if prog != nil {
					prog.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
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
		sesh.Error(err, "Unable to get file", "File not found: %q", fileID)
		return
	}

	if file.Private && file.UserID != userID {
		sesh.Error(ErrPrivateFileAccess, "Unable to get file", "File not found: %q", fileID)
		return
	}

	args := sesh.Command()
	if len(args) == 0 {
		h.DownloadFile(sesh, &file)
		return
	}

	if file.UserID != userID {
		sesh.Error(ErrOpOnNonOwnedFile, "Unable to perform operation on file", "You do not own %q, therefore you cannot perform an operation on it.", fileID)
		return
	}

	switch args[0] {
	case "rm":
		h.DeleteFile(sesh, &file)
	case "sign":
		h.SignFile(sesh, &file)
	default:
		sesh.Error(ErrUnknownCommand, "Unknown command", "Unknown command specified: %q", args[0])
	}
}

func (h *SessionHandler) DeleteFile(sesh *UserSession, file *db.File) {
	flags := DeleteFlags{}
	args := sesh.Command()[1:]
	if err := flags.Parse(sesh.Stderr(), args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Warn().Err(err).Msg("invalid user specified flags")
			flags.PrintDefaults()
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

	// this is a soft delete
	if err := h.DB.Delete(file).Error; err != nil {
		sesh.Error(err, "Unable to delete file", "There was an error deleting file: %q", file.ID)
		return
	}

	log.Info().Fields(map[string]interface{}{
		"file_id": file.ID,
		"user_id": file.UserID,
	}).Msg("file deleted")

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

func (h *SessionHandler) SignFile(sesh *UserSession, file *db.File) {
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

	noti := Notification{
		Color: styles.Colors.Cyan,
		Title: "Signed URL Generated 📝",
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}
	noti.Messagef("Expires in: %s", styles.C(styles.Colors.Yellow, expires.Format(time.RFC3339)))
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
			sesh.Error(err, "Unable to read file", "There was an error reading the file: %q", err.Error())
			return
		}

		size += uint64(n)
		content = append(content, buf[:n]...)

		if size > MaxUploadSize {
			sesh.Error(ErrFileTooLarge, "Unable to upload file", "File too large, max size is %s", humanize.Bytes(MaxUploadSize))
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

			file := db.File{
				Private: flags.Private,
				Content: content,
				Size:    size,
				UserID:  sesh.UserID(),
				Type:    renderer.DetectFileType(content, flags.Extension),
			}

			if err := h.DB.Create(&file).Error; err != nil {
				sesh.Error(err, "Unable to create file", "There was an error creating the file: %s", err.Error())
				return
			}

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

			httpAddr := h.Config.HTTP.External
			httpAddr.Path = fmt.Sprintf("/f/%s", file.ID)

			sshCommand := fmt.Sprintf("ssh %s%s@%s", FileRequestPrefix, file.ID, h.Config.SSH.External.Hostname())
			if sshPort := h.Config.SSH.External.Port(); sshPort != "22" {
				sshCommand += fmt.Sprintf(" -p %s", sshPort)
			}

			noti = Notification{
				Title:   "SSH 📠",
				Message: styles.C(styles.Colors.Yellow, sshCommand),
				WithStyle: func(s *lipgloss.Style) {
					s.MarginTop(1)
				},
			}
			noti.Render(sesh)

			url := lipgloss.NewStyle().
				Foreground(styles.Colors.Blue).
				Underline(true).
				Render(httpAddr.String())

			noti = Notification{
				Title:   "URL 🔗",
				Message: url,
				WithStyle: func(s *lipgloss.Style) {
					s.MarginTop(1)
				},
			}

			if file.Private {
				noti.Message = "<none> (requires a signed URL)"
			}

			noti.Render(sesh)

			return
		}
	}
}
