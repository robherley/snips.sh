package styles

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
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
