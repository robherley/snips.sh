package prompt

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/feedback"
)

// renameDialog sets or clears a file's human-readable name.
type renameDialog struct {
	textDialog
}

func newRenameDialog() *renameDialog {
	return &renameDialog{newTextDialog()}
}

func (d *renameDialog) title() string {
	return "rename"
}

func (d *renameDialog) question(file *snips.File) string {
	if file.Name != "" {
		return fmt.Sprintf("What do you want to rename %q to?\n(submit empty to remove the name)", file.Name)
	}
	return fmt.Sprintf("What do you want to name %q?", file.ID)
}

func (d *renameDialog) submit(e env) tea.Cmd {
	name := strings.TrimSpace(d.value())

	if name == "" {
		return d.removeName(e)
	}

	normalized, err := snips.NormalizeName(name)
	if err != nil {
		return SetPromptErrorCmd(err)
	}

	previous := e.file.Name
	e.file.Name = normalized
	if err := e.db.UpdateFile(e.ctx, e.file); err != nil {
		e.file.Name = previous
		if errors.Is(err, db.ErrNameTaken) {
			return SetPromptErrorCmd(fmt.Errorf("you already have a file named %q", normalized))
		}
		return SetPromptErrorCmd(err)
	}

	metrics.IncrCounter([]string{"file", "rename"}, 1)
	logger.From(e.ctx).Info("file renamed", "file", e.file.ID, "name", e.file.Name)

	msg := feedback.Success(fmt.Sprintf("file %q is now named %q", e.file.ID, e.file.Name))
	return tea.Batch(cmds.ReloadFiles(e.db, e.file.UserID), SetPromptFeedbackCmd(msg, true))
}

func (d *renameDialog) removeName(e env) tea.Cmd {
	if e.file.Name == "" {
		return SetPromptErrorCmd(errors.New("please enter a name"))
	}

	old := e.file.Name
	e.file.Name = ""
	if err := e.db.UpdateFile(e.ctx, e.file); err != nil {
		e.file.Name = old
		return SetPromptErrorCmd(err)
	}

	metrics.IncrCounter([]string{"file", "rename"}, 1)
	logger.From(e.ctx).Info("file name removed", "file", e.file.ID, "old_name", old)

	msg := feedback.Success(fmt.Sprintf("file %q is no longer named %q", e.file.ID, old))
	return tea.Batch(cmds.ReloadFiles(e.db, e.file.UserID), SetPromptFeedbackCmd(msg, true))
}
