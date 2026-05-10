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

type Settings struct {
	db          db.DB
	user        *snips.User
	fingerprint string

	width  int
	height int

	themeIndex int
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
		themeIndex:  themeIndexOf(user.ThemeColor),
	}
}

func themeIndexOf(name string) int {
	for i, opt := range styles.ThemeOptions {
		if opt.Name == name {
			return i
		}
	}
	return 0 // default to first option
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
			return s.save()
		}
	}
	return s, nil
}

func (s Settings) save() (tea.Model, tea.Cmd) {
	name := styles.ThemeOptions[s.themeIndex].Name
	s.user.ThemeColor = name
	if err := s.db.UpdateUser(context.Background(), s.user); err != nil {
		s.feedback = "failed to save: " + err.Error()
		s.feedbackOK = false
		return s, nil
	}

	s.feedback = "theme set to " + name
	s.feedbackOK = true

	return s, func() tea.Msg { return msgs.ThemeChanged{Color: styles.ThemeOptions[s.themeIndex].Color} }
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
		"",
		labelStyle.Render("theme color"),
	}

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

	rows = append(rows, "", helpStyle.Render("↑/↓ navigate · ↵ apply"))

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

func (s Settings) Keys() help.KeyMap {
	return keys
}

func (s Settings) IsCapturing() bool {
	return false
}
