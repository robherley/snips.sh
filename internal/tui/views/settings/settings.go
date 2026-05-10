package settings

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Settings struct {
	userID      string
	fingerprint string
	width       int
	height      int
}

func New(width, height int, userID, fingerprint string) Settings {
	return Settings{
		userID:      userID,
		fingerprint: fingerprint,
		width:       width,
		height:      height,
	}
}

func (s Settings) Init() tea.Cmd {
	return nil
}

func (s Settings) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		s.width, s.height = size.Width, size.Height
	}
	return s, nil
}

func (s Settings) View() tea.View {
	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Muted).
		Bold(true)
	valueStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.White)

	fields := []struct{ label, value string }{
		{"user id", s.userID},
		{"fingerprint", s.fingerprint},
	}

	parts := make([]string, 0, len(fields)*3)
	for _, f := range fields {
		parts = append(parts,
			labelStyle.Render(f.label),
			valueStyle.Render(f.value),
			"",
		)
	}

	content := lipgloss.NewStyle().
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Top, parts...))

	// fill the allocated height so the surrounding TUI's bottom bar stays anchored
	return tea.NewView(lipgloss.Place(s.width, s.height, lipgloss.Left, lipgloss.Top, content))
}

func (s Settings) Keys() help.KeyMap {
	return keys
}

func (s Settings) IsCapturing() bool {
	return false
}
