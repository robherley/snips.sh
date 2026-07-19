package prompt

import (
	"context"
	"image/color"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// Prompt hosts the active dialog: it renders the modal frame, question, and
// feedback, and routes input to the dialog in between.
type Prompt struct {
	ctx    context.Context
	cfg    *config.Config
	db     db.DB
	width  int
	height int
	theme  color.Color

	file     *snips.File
	dialog   dialog
	feedback string
	finished bool
}

// contentWidth caps modal content so long feedback and the extension selector
// wrap sensibly instead of stretching the window across the terminal.
func contentWidth(width int) int {
	return max(styles.ModalMinWidth, min(width-10, 76))
}

func New(ctx context.Context, cfg *config.Config, db db.DB, width, height int, theme color.Color) Prompt {
	return Prompt{
		ctx:    ctx,
		cfg:    cfg,
		db:     db,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (p Prompt) Init() tea.Cmd {
	return nil
}

func (p Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEnter {
			return p, p.submit()
		}
	case FeedbackMsg:
		p.feedback = msg.Feedback
		p.finished = msg.Finished
		return p, nil
	case KindSetMsg:
		// each open gets a fresh dialog, so no input state leaks between uses
		p.dialog = newDialog(msg.Kind, contentWidth(p.width))
		if p.dialog != nil {
			return p, p.dialog.init()
		}
		return p, nil
	case msgs.FileLoaded:
		p.file = msg.File
		return p, nil
	case msgs.PopView:
		p.dialog = nil
		p.feedback = ""
		p.finished = false
		return p, nil
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		if p.dialog != nil {
			p.dialog.resize(contentWidth(msg.Width))
		}
		return p, nil
	case msgs.ThemeChanged:
		p.theme = msg.Color
		return p, nil
	}

	// everything else (keystrokes, blinks, filter matches) belongs to the
	// active dialog's input model
	if p.dialog == nil {
		return p, nil
	}
	return p, p.dialog.update(msg)
}

func (p Prompt) submit() tea.Cmd {
	if p.finished || p.dialog == nil || p.file == nil {
		return nil
	}

	return p.dialog.submit(env{
		ctx:  p.ctx,
		cfg:  p.cfg,
		db:   p.db,
		file: p.file,
	})
}

func (p Prompt) View() tea.View {
	if p.file == nil || p.dialog == nil {
		return tea.NewView(lipgloss.Place(p.width, p.height, lipgloss.Left, lipgloss.Top, ""))
	}

	return tea.NewView(styles.ModalBody(p.theme, "options / "+p.dialog.title(), p.renderPrompt()))
}

func (p Prompt) Keys() help.KeyMap {
	return newKeyMap(p.finished)
}

func (p Prompt) IsCapturing() bool {
	return false
}

func (p Prompt) renderPrompt() string {
	question := lipgloss.NewStyle().
		Foreground(styles.Colors.White).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeftForeground(styles.Colors.Yellow).
		PaddingLeft(1).
		Render(p.dialog.question(p.file))

	pieces := []string{}

	if !p.finished {
		pieces = append(pieces,
			question,
			"",
			p.dialog.view(),
		)
	}

	if p.feedback != "" {
		if !p.finished {
			pieces = append(pieces, "")
		}
		pieces = append(pieces, p.feedback)
	}

	return lipgloss.JoinVertical(lipgloss.Top, pieces...)
}
