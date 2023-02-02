package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

type HeaderKind string

const (
	HeaderSuccess HeaderKind = "SUCCESS"
	HeaderError   HeaderKind = "ERROR"
)

func PrintHeader(sesh ssh.Session, kind HeaderKind, title string) {
	var color lipgloss.Color
	switch kind {
	case HeaderSuccess:
		color = lipgloss.Color("10")
	case HeaderError:
		color = lipgloss.Color("9")
	}

	statusRender := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Padding(0, 1).
		Render(string(kind))

	titleRender := lipgloss.NewStyle().
		Foreground(color).
		Align(lipgloss.Right).
		MarginLeft(1).
		Render(title)

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		statusRender,
		titleRender,
	)

	wish.Println(sesh, bar)
}
