package settings

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// themeView is the theme color picker page.
type themeView struct {
	deps
	index int
}

func newThemeView(d deps) themeView {
	return themeView{deps: d}
}

// themeIndexOf returns the index of the named theme option, defaulting to the
// first option.
func themeIndexOf(name string) int {
	for i, opt := range styles.ThemeOptions {
		if opt.Name == name {
			return i
		}
	}
	return 0
}

// enter resets the selection to the user's current theme.
func (m themeView) enter() themeView {
	m.index = themeIndexOf(m.user.ThemeColor)
	return m
}

func (m themeView) update(msg tea.KeyPressMsg) (themeView, result) {
	switch msg.String() {
	case "up", "k":
		if m.index > 0 {
			m.index--
		}
	case "down", "j":
		if m.index < len(styles.ThemeOptions)-1 {
			m.index++
		}
	case "enter":
		return m, m.save()
	case "esc":
		return m, result{back: true}
	case "q":
		// the view captures input on deeper pages, so quit needs handling here
		return m, result{quit: true}
	}
	return m, result{}
}

// save persists the selected theme and returns to the root page.
func (m themeView) save() result {
	opt := styles.ThemeOptions[m.index]

	prev := m.user.ThemeColor
	m.user.ThemeColor = opt.Name
	if err := m.db.UpdateUser(m.ctx, m.user); err != nil {
		m.user.ThemeColor = prev
		return result{back: true, fb: feedback.Error("failed to save: " + err.Error())}
	}

	return result{
		back: true,
		fb:   feedback.Success("theme set to " + opt.Name),
		cmd:  func() tea.Msg { return msgs.ThemeChanged{Color: opt.Color} },
	}
}

func (m themeView) rows() []string {
	rows := make([]string, 0, len(styles.ThemeOptions))
	for i, opt := range styles.ThemeOptions {
		swatch := lipgloss.NewStyle().Background(opt.Color).Render("   ")
		cursor := "  "
		nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
		if i == m.index {
			cursor = lipgloss.NewStyle().Foreground(opt.Color).Bold(true).Render("→ ")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		}
		rows = append(rows, cursor+swatch+"  "+nameStyle.Render(opt.Name))
	}
	return rows
}

// themeKeyMap is shown while navigating the theme color page.
type themeKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Esc   key.Binding
	Quit  key.Binding
}

func (k themeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Esc, k.Quit}
}

func (k themeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Enter, k.Esc, k.Quit},
	}
}

var themeKeys = themeKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "apply"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
