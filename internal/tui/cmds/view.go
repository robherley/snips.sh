package cmds

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/views"
)

func ChangeView(view views.View) tea.Cmd {
	return func() tea.Msg {
		return msgs.ChangeView{
			View: view,
		}
	}
}
