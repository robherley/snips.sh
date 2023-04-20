package prompt

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Help   key.Binding
	Quit   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Help, km.Escape, km.Enter}
}

func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Escape, km.Enter},
		{km.Help, km.Quit},
	}
}

var keys = keyMap{
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
