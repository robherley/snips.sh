package prompt

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

func SetPromptKindCmd(pk Kind) tea.Cmd {
	return func() tea.Msg {
		return PromptKindSetMsg{
			Kind: pk,
		}
	}
}

func SetPromptFeedbackCmd(feedback string, finished bool) tea.Cmd {
	return func() tea.Msg {
		return PromptFeedbackMsg{
			Feedback: feedback,
			Finished: finished,
		}
	}
}

func SetPromptErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return PromptFeedbackMsg{
			Feedback: styles.C(styles.Colors.Red, err.Error()),
			Finished: false,
		}
	}
}
