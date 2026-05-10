package cmds

import (
	tea "charm.land/bubbletea/v2"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/views"
)

func PushView(view views.Kind) tea.Cmd {
	return func() tea.Msg {
		return msgs.PushView{
			View: view,
		}
	}
}

func PopView() tea.Cmd {
	return func() tea.Msg {
		return msgs.PopView{}
	}
}
