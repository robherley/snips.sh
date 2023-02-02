package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
)

type Model struct {
	Term        string
	Width       int
	Height      int
	Time        time.Time
	UserID      string
	Fingerprint string
	Files       []db.File
}

type TimeMsg time.Time

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TimeMsg:
		m.Time = time.Time(msg)
	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) View() string {
	s := "👋 Welcome to snips.sh!\n"
	s += "🪪  You are user: %s\n"
	s += "🔑 Using key with fingerprint: %s\n"
	s += "📁 You have %d files\n"
	s += "🖥️  Your term is %s (x: %d, y: %d)\n"
	s += "⌚ Time: " + m.Time.Format(time.RFC1123) + "\n\n"
	s += "Press 'q' to quit\n"
	return fmt.Sprintf(s, m.UserID, m.Fingerprint, len(m.Files), m.Term, m.Width, m.Height)
}
