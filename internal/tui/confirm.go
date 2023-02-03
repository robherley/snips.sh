package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

func Confirm(sesh ssh.Session, f string, v ...interface{}) (bool, error) {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	prompt := style.Render(fmt.Sprintf(f, v...) + " [y/N] ")
	wish.Print(sesh, prompt)

	option := make([]byte, 1)
	n, err := sesh.Read(option)
	if err != nil {
		return false, err
	}

	if n == 0 {
		return false, nil
	}

	if option[0] == 'y' || option[0] == 'Y' {
		return true, nil
	}

	return false, nil
}
