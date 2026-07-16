package prompt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// visibilityDialog toggles a file between public and private after a y/n
// confirmation.
type visibilityDialog struct {
	textDialog
}

func newVisibilityDialog() *visibilityDialog {
	return &visibilityDialog{newTextDialog()}
}

func (d *visibilityDialog) title() string {
	return "toggle visibility"
}

func (d *visibilityDialog) question(file *snips.File) string {
	question := fmt.Sprintf("Do you want to make the file %q ", file.ID)
	if file.Private {
		question += "public"
	} else {
		question += "private"
	}
	return question + "?\n(y/n)"
}

func (d *visibilityDialog) submit(e env) tea.Cmd {
	switch d.answer() {
	case undecided:
		return SetPromptErrorCmd(errors.New("please specify yes or no"))
	case no:
		return cmds.PopView()
	}

	e.file.Private = !e.file.Private

	if err := e.db.UpdateFile(e.ctx, e.file); err != nil {
		return SetPromptErrorCmd(err)
	}

	metrics.IncrCounterWithLabels([]string{"file", "change", "private"}, 1, []metrics.Label{
		{Name: "new", Value: strconv.FormatBool(e.file.Private)},
	})
	logger.From(e.ctx).Info("updated file visibility", "file", e.file.ID, "private", e.file.Private)

	msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q is now %s", e.file.ID, e.file.Visibility()))
	return tea.Batch(cmds.ReloadFiles(e.db, e.file.UserID), SetPromptFeedbackCmd(msg, true))
}

type ynResult int

const (
	undecided ynResult = iota
	yes
	no
)

func (d *visibilityDialog) answer() ynResult {
	lower := strings.ToLower(d.value())
	if len(lower) == 0 {
		return undecided
	}

	switch lower[0] {
	case 'y':
		return yes
	case 'n':
		return no
	default:
		return undecided
	}
}
