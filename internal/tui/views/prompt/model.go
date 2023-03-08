package prompt

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
)

type Model struct {
	file *db.File

	textInput textinput.Model
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "placeholder"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	return Model{
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf(
		"temp question?\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}
