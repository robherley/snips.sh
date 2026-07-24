package cmds

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
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

func LoadFile(database *db.DB, id string) tea.Cmd {
	return func() tea.Msg {
		file, err := database.Files.Find(context.Background(), id)
		if err != nil {
			return msgs.Error{Err: err}
		}

		if file == nil {
			return msgs.Error{Err: errors.New("file not found")}
		}
		content, err := database.Files.GetContent(context.Background(), id)
		if err != nil {
			return msgs.Error{Err: err}
		}

		return msgs.FileLoaded{
			File:    file,
			Content: content,
		}
	}
}

func ReloadFiles(database *db.DB, userID string) tea.Cmd {
	return func() tea.Msg {
		files, err := database.Files.FindByUser(context.Background(), userID)
		if err != nil {
			return msgs.Error{Err: err}
		}

		return msgs.ReloadFiles{
			Files: files,
		}
	}
}
