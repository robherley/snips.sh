package prompt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Prompt struct {
	ctx      context.Context
	cfg      *config.Config
	db       db.DB
	width    int
	finished bool

	file              *snips.File
	pk                Kind
	textInput         textinput.Model
	extensionSelector list.Model
	feedback          string
}

func New(ctx context.Context, cfg *config.Config, db db.DB, width int) Prompt {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 20
	ti.Prompt = styles.BC(styles.Colors.Yellow, "> ")

	return Prompt{
		ctx:               ctx,
		cfg:               cfg,
		db:                db,
		textInput:         ti,
		extensionSelector: NewExtensionSelector(width),
		width:             width,
	}
}

func (m Prompt) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (m Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd      tea.Cmd
		commands []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, m.handleSubmit()
		}
	case PromptFeedbackMsg:
		m.feedback = msg.Feedback
		m.finished = msg.Finished
	case PromptKindSetMsg:
		m.pk = msg.Kind
	case msgs.FileLoaded:
		m.file = msg.File
	case msgs.PopView:
		m.reset()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.extensionSelector.SetWidth(msg.Width)
	}

	switch m.pk {
	case GenerateSignedURL, DeleteFile, ChangeVisibility:
		m.textInput, cmd = m.textInput.Update(msg)
		commands = append(commands, cmd)
	case ChangeExtension:
		m.extensionSelector, cmd = m.extensionSelector.Update(msg)
		commands = append(commands, cmd)
	}
	return m, tea.Batch(commands...)
}

func (m Prompt) View() string {
	if m.file == nil || m.pk == None {
		return ""
	}

	return m.renderPrompt()
}

func (m *Prompt) reset() {
	m.textInput.Reset()
	m.extensionSelector.ResetFilter()
	m.extensionSelector.ResetSelected()
	m.feedback = ""
	m.finished = false
}

func (m Prompt) renderPrompt() string {
	var question string
	switch m.pk {
	case ChangeExtension:
		question = "What extension do you want to change the file to?\n" + styles.C(styles.Colors.Muted, "(type / to filter)")
	case ChangeVisibility:
		question = fmt.Sprintf("Do you want to make the file %q ", m.file.ID)
		if m.file.Private {
			question += "public"
		} else {
			question += "private"
		}
		question += "?\n(y/n)"
	case GenerateSignedURL:
		question = fmt.Sprintf("How long do you want the signed url for %q to last for?\n%s", m.file.ID, styles.C(styles.Colors.Muted, "(e.g. 30s, 5m, 3h)"))
	case DeleteFile:
		question = fmt.Sprintf("Are you sure you want to delete %q?\nType the file ID to confirm.", m.file.ID)
	}

	question = lipgloss.NewStyle().
		Foreground(styles.Colors.White).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeftForeground(styles.Colors.Yellow).
		PaddingLeft(1).
		Render(question)

	var prompt string
	switch m.pk {
	case GenerateSignedURL, DeleteFile, ChangeVisibility:
		prompt = m.textInput.View()
	case ChangeExtension:
		prompt = m.extensionSelector.View()
	}

	pieces := []string{}

	if !m.finished {
		pieces = append(pieces,
			"",
			question,
			"",
			prompt,
			"",
		)
	}

	if m.feedback != "" {
		pieces = append(pieces, "", wordwrap.String(m.feedback, m.width))
	}

	pieces = append(pieces, "", styles.C(styles.Colors.Muted, "(esc to go back)"))

	return lipgloss.JoinVertical(lipgloss.Top, pieces...)
}

func (m Prompt) handleSubmit() tea.Cmd {
	log := logger.From(m.ctx)

	var commands []tea.Cmd

	switch m.pk {
	case ChangeVisibility:
		result := m.textInputYN()

		if result == Undecided {
			return SetPromptErrorCmd(errors.New("please specify yes or no"))
		}

		if result == No {
			return cmds.PopView()
		}

		m.file.Private = !m.file.Private

		err := m.db.UpdateFile(m.ctx, m.file)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		log.Info().Str("file", m.file.ID).Bool("private", m.file.Private).Msg("updated file visibility")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q is now %s", m.file.ID, m.file.Visibility()))
		commands = append(commands, cmds.ReloadFiles(m.db, m.file.UserID), SetPromptFeedbackCmd(msg, true))

	case ChangeExtension:
		item := m.extensionSelector.SelectedItem().(selectorItem)

		m.file.Type = item.name

		err := m.db.UpdateFile(m.ctx, m.file)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		log.Info().Str("file", m.file.ID).Str("type", m.file.Type).Msg("updated file type")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q extension set to %q", m.file.ID, item.name))
		commands = append(commands, cmds.ReloadFiles(m.db, m.file.UserID), SetPromptFeedbackCmd(msg, true))
	case GenerateSignedURL:
		dur, err := time.ParseDuration(m.textInput.Value())
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		if dur <= 0 {
			return SetPromptErrorCmd(errors.New("duration must be greater than 0"))
		}

		url, expires := m.file.GetSignedURL(m.cfg, dur)
		log.Info().Str("file_id", m.file.ID).Time("expires_at", expires).Msg("private file signed")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("%s\n\nexpires at: %s", url.String(), expires.Format(time.RFC3339)))
		commands = append(commands, SetPromptFeedbackCmd(msg, true))
	case DeleteFile:
		if cmd := m.validateInputIsFileID(); cmd != nil {
			return cmd
		}

		err := m.db.DeleteFile(m.ctx, m.file.ID)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		log.Info().Str("file_id", m.file.ID).Msg("file deleted")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q deleted", m.file.ID))
		commands = append(commands, cmds.ReloadFiles(m.db, m.file.UserID), SetPromptFeedbackCmd(msg, true))
	default:
		return nil
	}

	return tea.Batch(commands...)
}

func (m Prompt) validateInputIsFileID() tea.Cmd {
	if m.textInput.Value() != m.file.ID {
		return SetPromptErrorCmd(errors.New("please specify the file id to confirm"))
	}

	return nil
}

type YNResult int

const (
	Undecided YNResult = iota
	Yes
	No
)

func (m Prompt) textInputYN() YNResult {
	lower := strings.ToLower(m.textInput.Value())
	if len(lower) == 0 {
		return Undecided
	}

	switch lower[0] {
	case 'y':
		return Yes
	case 'n':
		return No
	default:
		return Undecided
	}
}
