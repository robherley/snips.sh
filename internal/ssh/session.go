package ssh

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"

	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/tui/styles"
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

	noti := Notification{
		Color: styles.Colors.Red,
		WithStyle: func(s *lipgloss.Style) {
			s.MarginTop(1)
		},
	}

	noti.Titlef("%s ⛔", title)
	noti.Messagef(f, v...)

	if sesh.RequestID() != "" {
		noti.Message += "\nRequest ID: " + sesh.RequestID()
	}

	noti.Render(sesh)
	_ = sesh.Exit(1)
}

func (sesh *UserSession) IsPTY() bool {
	_, _, isPty := sesh.Pty()
	return isPty
}
