package settings

import (
	"context"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type itemKind int

const (
	// themePicker expands into the styles.ThemeOptions list when entered.
	themePicker itemKind = iota
)

// item is a single navigable row in the settings view.
type item struct {
	kind itemKind
}

// section groups related items under a header.
type section struct {
	title string
	items []item
}

// newSections lists every setting; add new sections (or items) here.
func newSections() []section {
	return []section{
		{title: "theme color", items: []item{{kind: themePicker}}},
	}
}

type Settings struct {
	db          db.DB
	user        *snips.User
	fingerprint string

	width  int
	height int

	sections   []section
	cursor     int  // top-level item index, flat across all sections
	focused    bool // navigating inside the theme picker
	themeIndex int  // selection within the theme picker while focused
	feedback   string
	feedbackOK bool
}

func New(width, height int, database db.DB, user *snips.User, fingerprint string) Settings {
	return Settings{
		db:          database,
		user:        user,
		fingerprint: fingerprint,
		width:       width,
		height:      height,
		sections:    newSections(),
	}
}

// items flattens all sections' items in display order.
func (s Settings) items() []item {
	items := []item{}
	for _, sec := range s.sections {
		items = append(items, sec.items...)
	}
	return items
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

func (s Settings) Init() tea.Cmd {
	return nil
}

func (s Settings) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		return s, nil
	case tea.KeyPressMsg:
		if s.focused {
			return s.updateThemePicker(msg)
		}

		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.items())-1 {
				s.cursor++
			}
		case "enter":
			return s.activate()
		}
	}
	return s, nil
}

// activate enters the item under the cursor.
func (s Settings) activate() (tea.Model, tea.Cmd) {
	if s.items()[s.cursor].kind == themePicker {
		s.focused = true
		s.themeIndex = themeIndexOf(s.user.ThemeColor)
		s.feedback = ""
	}
	return s, nil
}

func (s Settings) updateThemePicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.themeIndex > 0 {
			s.themeIndex--
		}
	case "down", "j":
		if s.themeIndex < len(styles.ThemeOptions)-1 {
			s.themeIndex++
		}
	case "enter":
		s.focused = false
		return s.saveTheme(s.themeIndex)
	case "esc":
		s.focused = false
	case "q":
		// the view captures input while focused, so quit needs handling here
		return s, tea.Quit
	}
	return s, nil
}

func (s Settings) saveTheme(index int) (tea.Model, tea.Cmd) {
	name := styles.ThemeOptions[index].Name
	s.user.ThemeColor = name
	if err := s.db.UpdateUser(context.Background(), s.user); err != nil {
		s.feedback = "failed to save: " + err.Error()
		s.feedbackOK = false
		return s, nil
	}

	s.feedback = "theme set to " + name
	s.feedbackOK = true

	return s, func() tea.Msg { return msgs.ThemeChanged{Color: styles.ThemeOptions[index].Color} }
}

func (s Settings) View() tea.View {
	labelStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(styles.Colors.White)
	helpStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)

	rows := []string{
		labelStyle.Render("user id"),
		valueStyle.Render(s.user.ID),
		"",
		labelStyle.Render("fingerprint"),
		valueStyle.Render(s.fingerprint),
	}

	flat := 0
	for _, sec := range s.sections {
		rows = append(rows, "", labelStyle.Render(sec.title))
		for _, it := range sec.items {
			rows = append(rows, s.renderItem(it, flat == s.cursor)...)
			flat++
		}
	}

	if s.focused {
		rows = append(rows, "", helpStyle.Render("↑/↓ navigate · ↵ apply · esc back"))
	}

	if s.feedback != "" {
		c := styles.Colors.Green
		if !s.feedbackOK {
			c = styles.Colors.Red
		}
		rows = append(rows, "", lipgloss.NewStyle().Foreground(c).Render(s.feedback))
	}

	body := lipgloss.NewStyle().
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Top, rows...))

	return tea.NewView(lipgloss.Place(s.width, s.height, lipgloss.Left, lipgloss.Top, body))
}

// renderItem renders an item's rows. Items render a single collapsed row
// until entered; the theme picker expands into the full option list while
// focused.
func (s Settings) renderItem(it item, selected bool) []string {
	if it.kind != themePicker {
		return nil
	}

	if s.focused && selected {
		return s.renderThemeOptions()
	}

	opt := styles.ThemeOptions[themeIndexOf(s.user.ThemeColor)]
	swatch := lipgloss.NewStyle().Background(opt.Color).Render("   ")
	cursor := "  "
	nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
	hint := ""
	if selected {
		cursor = lipgloss.NewStyle().Foreground(opt.Color).Bold(true).Render("→ ")
		nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		hint = lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render("  ↵ change")
	}
	return []string{cursor + swatch + "  " + nameStyle.Render(opt.Name) + hint}
}

func (s Settings) renderThemeOptions() []string {
	rows := make([]string, 0, len(styles.ThemeOptions))
	for i, opt := range styles.ThemeOptions {
		swatch := lipgloss.NewStyle().Background(opt.Color).Render("   ")
		cursor := "  "
		nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
		if i == s.themeIndex {
			cursor = lipgloss.NewStyle().Foreground(opt.Color).Bold(true).Render("→ ")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		}
		rows = append(rows, cursor+swatch+"  "+nameStyle.Render(opt.Name))
	}
	return rows
}

func (s Settings) Keys() help.KeyMap {
	if s.focused {
		return pickerKeys
	}
	return keys
}

func (s Settings) IsCapturing() bool {
	// while inside the theme picker, keys (esc, q, ...) belong to this view
	// rather than the surrounding TUI
	return s.focused
}
