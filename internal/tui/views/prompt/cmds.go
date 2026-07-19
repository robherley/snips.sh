package prompt

import (
	tea "charm.land/bubbletea/v2"
	"github.com/robherley/snips.sh/internal/tui/styles"
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

func SetPromptFeedbackCmd(feedback string, finished bool) tea.Cmd {
	return func() tea.Msg {
		return FeedbackMsg{
			Feedback: feedback,
			Finished: finished,
		}
	}
}

func SetPromptErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return FeedbackMsg{
			Feedback: styles.C(styles.Colors.Red, err.Error()),
			Finished: false,
		}
	}
}

func SelectorInitCmd() tea.Msg {
	return SelectorInitMsg{}
}
