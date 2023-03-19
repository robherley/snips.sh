package cmds

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
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

func LoadFile(database db.DB, id, userID string) tea.Cmd {
	return func() tea.Msg {
		file, err := database.FileForUser(id, userID)
		if err != nil {
			return msgs.Error{Err: err}
		}

		if file == nil {
			return msgs.Error{Err: errors.New("file not found")}
		}

		return msgs.FileLoaded{
			File: file,
		}
	}
}
