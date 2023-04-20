package styles

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

var (
	Help = help.Styles{
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		Ellipsis:       sepStyle.Copy(),
		FullKey:        keyStyle.Copy(),
		FullDesc:       descStyle.Copy(),
		FullSeparator:  sepStyle.Copy(),
	}

	keyStyle  = lipgloss.NewStyle().Bold(true).Foreground(Colors.White)
	descStyle = lipgloss.NewStyle().Foreground(Colors.Muted)
	sepStyle  = lipgloss.NewStyle().Foreground(Colors.Muted)
)
