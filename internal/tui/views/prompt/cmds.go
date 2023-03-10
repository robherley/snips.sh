package prompt

import tea "github.com/charmbracelet/bubbletea"

func SetPromptKindCmd(pk Kind) tea.Cmd {
	return func() tea.Msg {
		return PromptKindSetMsg{
			Kind: pk,
		}
	}
}

func SetPromptErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return PromptError{
			Err: err,
		}
	}
}
