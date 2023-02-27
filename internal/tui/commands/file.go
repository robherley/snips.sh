package commands

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/tui/messages"
)

func GetFile(database *db.DB, id, userID string) tea.Cmd {
	return func() tea.Msg {
		file := db.File{}
		if err := database.Find(&file, "id = ? AND user_id = ?", id, userID).Error; err != nil {
			return messages.Error{Err: err}
		}

		return messages.FileLoaded{
			File: &file,
		}
	}
}
