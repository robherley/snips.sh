package prompt

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
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
	kind              Kind
	textInput         textinput.Model
	textarea          textarea.Model
	extensionSelector list.Model
	feedback          string
}

func New(ctx context.Context, cfg *config.Config, db db.DB, width int) Prompt {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 20
	ti.Prompt = styles.BC(styles.Colors.Yellow, "> ")

	ta := textarea.New()
	ta.SetWidth(width - 4) // Leave some margin
	ta.SetHeight(10)
	ta.CharLimit = 0 // No character limit for content editing
	ta.Prompt = ""
	ta.ShowLineNumbers = true

	return Prompt{
		ctx:               ctx,
		cfg:               cfg,
		db:                db,
		textInput:         ti,
		textarea:          ta,
		extensionSelector: NewExtensionSelector(width),
		width:             width,
	}
}

func (p Prompt) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (p Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd      tea.Cmd
		commands []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.kind == EditContent {
			// For content editing, use Ctrl+S to save
			if msg.Type == tea.KeyCtrlS {
				return p, p.handleSubmit()
			}
		} else {
			// For other prompts, use Enter to submit
			if msg.Type == tea.KeyEnter {
				return p, p.handleSubmit()
			}
		}
	case FeedbackMsg:
		p.feedback = msg.Feedback
		p.finished = msg.Finished
	case KindSetMsg:
		p.kind = msg.Kind
		if msg.Kind == ChangeExtension {
			commands = append(commands, SelectorInitCmd)
		}
		if msg.Kind == EditContent && p.file != nil {
			// Load file content into textarea
			content, err := p.file.GetContent()
			if err == nil {
				p.textarea.SetValue(string(content))
				p.textarea.Focus()
			}
		}
	case msgs.FileLoaded:
		p.file = msg.File
	case msgs.PopView:
		p.reset()
		return p, nil
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.extensionSelector.SetWidth(msg.Width)
		p.textarea.SetWidth(msg.Width - 4) // Leave some margin
	case SelectorInitMsg:
		// bit of a hack to get the extension selector to filter on init
		p.extensionSelector, cmd = p.extensionSelector.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'/'},
		})
		return p, cmd
	}

	switch p.kind {
	case GenerateSignedURL, DeleteFile, ChangeVisibility:
		p.textInput, cmd = p.textInput.Update(msg)
		commands = append(commands, cmd)
	case ChangeExtension:
		p.extensionSelector, cmd = p.extensionSelector.Update(msg)
		commands = append(commands, cmd)
	case EditContent:
		p.textarea, cmd = p.textarea.Update(msg)
		commands = append(commands, cmd)
	}
	return p, tea.Batch(commands...)
}

func (p Prompt) View() string {
	if p.file == nil || p.kind == None {
		return ""
	}

	return p.renderPrompt()
}

func (p Prompt) Keys() help.KeyMap {
	return newKeyMap(p.finished)
}

func (p *Prompt) reset() {
	p.textInput.Reset()
	p.textarea.Reset()
	p.extensionSelector.ResetFilter()
	p.extensionSelector.ResetSelected()
	p.feedback = ""
	p.finished = false
}

func (p Prompt) renderPrompt() string {
	var question string
	switch p.kind {
	case ChangeExtension:
		question = "What extension do you want to change the file to?"
	case ChangeVisibility:
		question = fmt.Sprintf("Do you want to make the file %q ", p.file.ID)
		if p.file.Private {
			question += "public"
		} else {
			question += "private"
		}
		question += "?\n(y/n)"
	case GenerateSignedURL:
		question = fmt.Sprintf("How long do you want the signed url for %q to last for?\n%s", p.file.ID, styles.C(styles.Colors.Muted, "(e.g. 30s, 5m, 3h)"))
	case DeleteFile:
		question = fmt.Sprintf("Are you sure you want to delete %q?\nType the file ID to confirm.", p.file.ID)
	case EditContent:
		question = fmt.Sprintf("Edit content for %q:\n%s", p.file.ID, styles.C(styles.Colors.Muted, "(Press Ctrl+S to save, Esc to cancel)"))
	}

	question = lipgloss.NewStyle().
		Foreground(styles.Colors.White).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeftForeground(styles.Colors.Yellow).
		PaddingLeft(1).
		Render(question)

	var prompt string
	switch p.kind {
	case GenerateSignedURL, DeleteFile, ChangeVisibility:
		prompt = p.textInput.View()
	case ChangeExtension:
		prompt = p.extensionSelector.View()
	case EditContent:
		prompt = p.textarea.View()
	}

	pieces := []string{}

	if !p.finished {
		pieces = append(pieces,
			"",
			question,
			"",
			prompt,
			"",
		)
	}

	if p.feedback != "" {
		pieces = append(pieces, "", wrap.String(p.feedback, p.width), "")
	}

	return lipgloss.JoinVertical(lipgloss.Top, pieces...)
}

func (p Prompt) handleSubmit() tea.Cmd {
	log := logger.From(p.ctx)

	if p.finished {
		return nil
	}

	var commands []tea.Cmd

	switch p.kind {
	case ChangeVisibility:
		result := p.textInputYN()

		if result == Undecided {
			return SetPromptErrorCmd(errors.New("please specify yes or no"))
		}

		if result == No {
			return cmds.PopView()
		}

		p.file.Private = !p.file.Private

		err := p.db.UpdateFile(p.ctx, p.file)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		metrics.IncrCounterWithLabels([]string{"file", "change", "private"}, 1, []metrics.Label{
			{Name: "new", Value: strconv.FormatBool(p.file.Private)},
		})
		log.Info().Str("file", p.file.ID).Bool("private", p.file.Private).Msg("updated file visibility")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q is now %s", p.file.ID, p.file.Visibility()))
		commands = append(commands, cmds.ReloadFiles(p.db, p.file.UserID), SetPromptFeedbackCmd(msg, true))

	case ChangeExtension:
		item := p.extensionSelector.SelectedItem().(selectorItem)
		old := p.file.Type
		p.file.Type = item.name

		err := p.db.UpdateFile(p.ctx, p.file)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		metrics.IncrCounterWithLabels([]string{"file", "change", "type"}, 1, []metrics.Label{
			{Name: "old", Value: old},
			{Name: "new", Value: p.file.Type},
		})
		log.Info().Str("file", p.file.ID).Str("old_type", old).Str("new_type", p.file.Type).Msg("updated file type")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q extension set to %q", p.file.ID, item.name))
		commands = append(commands, cmds.ReloadFiles(p.db, p.file.UserID), SetPromptFeedbackCmd(msg, true))
	case GenerateSignedURL:
		dur, err := time.ParseDuration(p.textInput.Value())
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		if dur <= 0 {
			return SetPromptErrorCmd(errors.New("duration must be greater than 0"))
		}

		url, expires := p.file.GetSignedURL(p.cfg, dur)

		metrics.IncrCounter([]string{"file", "sign"}, 1)
		log.Info().Str("file_id", p.file.ID).Time("expires_at", expires).Msg("private file signed")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("%s\n\nexpires at: %s", url.String(), expires.Format(time.RFC3339)))
		commands = append(commands, SetPromptFeedbackCmd(msg, true))
	case DeleteFile:
		if cmd := p.validateInputIsFileID(); cmd != nil {
			return cmd
		}

		err := p.db.DeleteFile(p.ctx, p.file.ID)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		metrics.IncrCounter([]string{"file", "delete"}, 1)
		log.Info().Str("file_id", p.file.ID).Msg("file deleted")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q deleted", p.file.ID))
		commands = append(commands, cmds.ReloadFiles(p.db, p.file.UserID), SetPromptFeedbackCmd(msg, true))
	case EditContent:
		content := p.textarea.Value()
		
		// Update the file content
		p.file.RawContent = []byte(content)
		p.file.Size = uint64(len(content))

		err := p.db.UpdateFile(p.ctx, p.file)
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		metrics.IncrCounter([]string{"file", "edit"}, 1)
		log.Info().Str("file_id", p.file.ID).Int("new_size", len(content)).Msg("file content updated")

		msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q content updated", p.file.ID))
		commands = append(commands, cmds.ReloadFiles(p.db, p.file.UserID), SetPromptFeedbackCmd(msg, true))
	default:
		return nil
	}

	return tea.Batch(commands...)
}

func (p Prompt) validateInputIsFileID() tea.Cmd {
	if p.textInput.Value() != p.file.ID {
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

func (p Prompt) textInputYN() YNResult {
	lower := strings.ToLower(p.textInput.Value())
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
