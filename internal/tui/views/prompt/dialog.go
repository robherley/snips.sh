package prompt

import (
	"context"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// dialog is a single prompt flow hosted by the Prompt view. Each dialog owns
// its input model and performs its action on submit; the host renders the
// modal frame, question, and feedback around it.
type dialog interface {
	// title is the breadcrumb segment shown in the modal header.
	title() string
	// question is shown above the input.
	question(file *snips.File) string
	// init returns the dialog's startup command, if any.
	init() tea.Cmd
	// update forwards a message to the dialog's input model.
	update(msg tea.Msg) tea.Cmd
	// view renders the dialog's input model.
	view() string
	// resize adjusts the dialog to the available content width.
	resize(width int)
	// submit validates the input and performs the dialog's action.
	submit(e env) tea.Cmd
}

// env is the shared context a dialog needs to perform its action.
type env struct {
	ctx  context.Context
	cfg  *config.Config
	db   db.DB
	file *snips.File
}

// newDialog builds a fresh dialog for the kind, or nil for None.
func newDialog(kind Kind, width int) dialog {
	switch kind {
	case ChangeExtension:
		return newExtensionDialog(width)
	case ChangeVisibility:
		return newVisibilityDialog()
	case GenerateSignedURL:
		return newSignedURLDialog()
	case DeleteFile:
		return newDeleteDialog()
	default:
		return nil
	}
}

// textDialog is the base for dialogs driven by a single text input.
type textDialog struct {
	input textinput.Model
}

func newTextDialog() textDialog {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 255
	ti.SetWidth(20)
	ti.Prompt = styles.BC(styles.Colors.Yellow, "> ")
	return textDialog{input: ti}
}

func (d *textDialog) init() tea.Cmd {
	return textinput.Blink
}

func (d *textDialog) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

func (d *textDialog) view() string {
	return d.input.View()
}

func (d *textDialog) resize(int) {}

func (d *textDialog) value() string {
	return d.input.Value()
}
