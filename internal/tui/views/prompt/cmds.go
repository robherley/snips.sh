package prompt

import (
	tea "charm.land/bubbletea/v2"
	"github.com/robherley/snips.sh/internal/tui/feedback"
)

// SetPromptKindCmd opens the dialog for the kind. The breadcrumb is the modal
// title's prefix, naming where the prompt was opened from ("" for none).
func SetPromptKindCmd(pk Kind, breadcrumb string) tea.Cmd {
	return func() tea.Msg {
		return KindSetMsg{
			Kind:       pk,
			Breadcrumb: breadcrumb,
		}
	}
}

func SetPromptFeedbackCmd(fb feedback.Feedback, finished bool) tea.Cmd {
	return func() tea.Msg {
		return FeedbackMsg{
			Feedback: fb,
			Finished: finished,
		}
	}
}

func SetPromptErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return FeedbackMsg{
			Feedback: feedback.Error(err.Error()),
			Finished: false,
		}
	}
}

func SelectorInitCmd() tea.Msg {
	return SelectorInitMsg{}
}
