package prompt

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// deleteDialog deletes a file after the user types its ID to confirm.
type deleteDialog struct {
	textDialog
}

func newDeleteDialog() *deleteDialog {
	return &deleteDialog{newTextDialog()}
}

func (d *deleteDialog) title() string {
	return "delete file"
}

func (d *deleteDialog) question(file *snips.File) string {
	return fmt.Sprintf("Are you sure you want to delete %q?\nType the file ID to confirm.", file.ID)
}

func (d *deleteDialog) submit(e env) tea.Cmd {
	if d.value() != e.file.ID {
		return SetPromptErrorCmd(errors.New("please specify the file id to confirm"))
	}

	if err := e.db.DeleteFile(e.ctx, e.file.ID); err != nil {
		return SetPromptErrorCmd(err)
	}

	metrics.IncrCounter([]string{"file", "delete"}, 1)
	logger.From(e.ctx).Info("file deleted", "file_id", e.file.ID)

	msg := styles.C(styles.Colors.Green, fmt.Sprintf("file %q deleted", e.file.ID))
	return tea.Batch(cmds.ReloadFiles(e.db, e.file.UserID), SetPromptFeedbackCmd(msg, true))
}
