package ssh

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
)

type model struct {
	term        string
	width       int
	height      int
	time        time.Time
	userID      string
	fingerprint string
	files       []db.File
}

type timeMsg time.Time

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeMsg:
		m.time = time.Time(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	s := "👋 Welcome to snips.sh!\n"
	s += "🪪  You are user: %s\n"
	s += "🔑 Using key with fingerprint: %s\n"
	s += "📁 You have %d files\n"
	s += "🖥️  Your term is %s (x: %d, y: %d)\n"
	s += "⌚ Time: " + m.time.Format(time.RFC1123) + "\n\n"
	s += "Press 'q' to quit\n"
	return fmt.Sprintf(s, m.userID, m.fingerprint, len(m.files), m.term, m.width, m.height)
}
