package ssh

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/tui"
)

type UserSession struct {
	ssh.Session
}

func (sesh *UserSession) UserID() string {
	return sesh.Context().Value(UserIDContextKey).(string)
}

func (sesh *UserSession) PublicKeyFingerprint() string {
	return sesh.Context().Value(FingerprintContextKey).(string)
}

func (sesh *UserSession) RequestID() string {
	return sesh.Context().Value(RequestIDContextKey).(string)
}

func (sesh *UserSession) IsFileRequest() bool {
	return strings.HasPrefix(sesh.User(), FileRequestPrefix)
}

func (sesh *UserSession) RequestedFileID() string {
	return strings.TrimPrefix(sesh.User(), FileRequestPrefix)
}

func (sesh *UserSession) Error(err error, title string, f string, v ...interface{}) {
	log := logger.From(sesh.Context())
	log.Error().Err(err).Msg(title)
	tui.PrintHeader(sesh, tui.HeaderError, title)
	wish.Errorf(sesh, f, v...)
	wish.Errorln(sesh)
	if sesh.RequestID() != "" {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("239"))
		wish.Errorln(sesh, style.Render("Request ID: "+sesh.RequestID()))
	}
}

func (sesh *UserSession) IsPTY() bool {
	_, _, isPty := sesh.Pty()
	return isPty
}
