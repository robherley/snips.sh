package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

type Notification struct {
	Color     lipgloss.Color
	Title     string
	Message   string
	WithStyle func(s *lipgloss.Style)
}

func (n *Notification) Titlef(format string, v ...interface{}) {
	n.Title = fmt.Sprintf(format, v...)
}

func (n *Notification) Messagef(format string, v ...interface{}) {
	n.Message = fmt.Sprintf(format, v...)
}

func (n *Notification) Render(sesh ssh.Session) {
	noti := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(n.Color).
		BorderLeft(true).
		PaddingLeft(1)

	if n.WithStyle != nil {
		n.WithStyle(&noti)
	}

	titleRender := lipgloss.NewStyle().
		Foreground(n.Color).
		Bold(true).
		Render(n.Title)

	messageRender := lipgloss.NewStyle().
		Foreground(Colors.Muted).
		Render(n.Message)

	notiRender := noti.Render(lipgloss.JoinVertical(lipgloss.Top, titleRender, messageRender))
	wish.Println(sesh, notiRender)
}
