package prompt

import (
	tea "charm.land/bubbletea/v2"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

func SetPromptKindCmd(pk Kind) tea.Cmd {
	return func() tea.Msg {
		return KindSetMsg{
			Kind: pk,
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
