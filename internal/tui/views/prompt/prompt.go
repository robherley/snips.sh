package prompt

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/models"
	"github.com/robherley/snips.sh/internal/tui/msgs"
)

type Prompt struct {
	file *models.File
	db   db.DB

	pk        Kind
	textInput textinput.Model
	err       error
}

func New(db db.DB) Prompt {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 20
	return Prompt{
		db:        db,
		textInput: ti,
	}
}

func (m Prompt) Init() tea.Cmd {
	return textinput.Blink
}

func (m Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, m.handleSubmit()
		}
	case PromptError:
		m.err = msg.Err
	case PromptKindSetMsg:
		m.pk = msg.Kind
	case msgs.FileLoaded:
		m.file = msg.File
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Prompt) View() string {
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

	str := fmt.Sprintf(
		"\n%s\n\n%s\n\n%s",
		prompt,
		m.textInput.View(),
		"(esc to go back)",
	) + "\n"

	if m.err != nil {
		str += fmt.Sprintf("\n%s\n", m.err.Error())
	}

	return str
}

func (m Prompt) handleSubmit() tea.Cmd {
	var cmds []tea.Cmd
	if m.err != nil {
		// reset the error
		cmds = append(cmds, SetPromptErrorCmd(nil))
	}

	switch m.pk {
	case ChangeVisibility:
		if cmd := m.validateInputIsFileID(); cmd != nil {
			return cmd
		}

		println("changing visibility of", m.file.ID, "to", !m.file.Private)
	case ChangeExtension:
		println("changing extension of", m.file.ID, "to", m.textInput.Value())
	case GenerateSignedURL:
		dur, err := time.ParseDuration(m.textInput.Value())
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		if dur <= 0 {
			return SetPromptErrorCmd(errors.New("duration must be greater than 0"))
		}

		println("generating signed url for", m.file.ID, "for", dur.String())
	case DeleteFile:
		if cmd := m.validateInputIsFileID(); cmd != nil {
			return cmd
		}

		println("delete file", m.file.ID)
	}

	return tea.Batch(cmds...)
}

func (m Prompt) validateInputIsFileID() tea.Cmd {
	if m.textInput.Value() != m.file.ID {
		return SetPromptErrorCmd(errors.New("please specify the file id to confirm"))
	}

	return nil
}
