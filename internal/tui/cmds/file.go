package cmds

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/models"
	"github.com/robherley/snips.sh/internal/tui/msgs"
)

func SelectFile(id string) tea.Cmd {
	return func() tea.Msg {
		return msgs.FileSelected{
			ID: id,
		}
	}
}

func DeselectFile() tea.Cmd {
	return func() tea.Msg {
		return msgs.FileDeselected{}
	}
}

func GetFile(database *db.DB, id, userID string) tea.Cmd {
	return func() tea.Msg {
		file := models.File{}
		if err := database.Find(&file, "id = ? AND user_id = ?", id, userID).Error; err != nil {
			return msgs.Error{Err: err}
		}

		return msgs.FileLoaded{
			File: &file,
		}
	}
}
