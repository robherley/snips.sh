package ssh

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/wish/v2"
	"github.com/charmbracelet/ssh"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Notification struct {
	Color     color.Color
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
		Foreground(styles.Colors.Muted).
		Render(n.Message)

	notiRender := noti.Render(lipgloss.JoinVertical(lipgloss.Top, titleRender, messageRender))
	wish.Println(sesh, notiRender)
}
