package styles

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

var (
	keyStyle  = lipgloss.NewStyle().Bold(true).Foreground(Colors.White)
	descStyle = lipgloss.NewStyle().Foreground(Colors.Muted)
	sepStyle  = lipgloss.NewStyle().Foreground(Colors.Muted)

	Help = help.Styles{
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		Ellipsis:       sepStyle,
		FullKey:        keyStyle,
		FullDesc:       descStyle,
		FullSeparator:  sepStyle,
	}
)
