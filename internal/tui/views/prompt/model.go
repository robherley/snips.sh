package prompt

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/tui/msgs"
)

type Model struct {
	file *db.File

	pk        Kind
	textInput textinput.Model
}

func New() Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 255
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			println(m.textInput.Value())
		}
	case PromptKindSetMsg:
		m.pk = msg.Kind
	case msgs.FileLoaded:
		m.file = msg.File
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.file == nil || m.pk == None {
		return ""
	}

	var prompt string
	switch m.pk {
	case ChangeExtension:
		prompt = "what would you like to change the extension to?\n(e.g. js, ruby, markdown)"
	case ChangeVisibility:
		prompt = fmt.Sprintf("do you want to make the file %q ", m.file.ID)
		if m.file.Private {
			prompt += "public"
		} else {
			prompt += "private"
		}
		prompt += "?\ntype the file id to confirm"
	case GenerateSignedURL:
		prompt = fmt.Sprintf("how long do you want the signed url for %q to last for?\n(e.g. 30s, 5m, 3h)", m.file.ID)
	case DeleteFile:
		prompt = fmt.Sprintf("are you sure you want to delete %q?\ntype the file id to confirm", m.file.ID)
	}

	return fmt.Sprintf(
		"\n%s\n\n%s\n\n%s",
		prompt,
		m.textInput.View(),
		"(esc to go back)",
	) + "\n"
}
