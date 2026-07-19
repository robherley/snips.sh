package prompt

import "charm.land/bubbles/v2/key"

type keyMap struct {
	submitted bool

	Help   key.Binding
	Quit   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	if km.submitted {
		return []key.Binding{km.Help, km.Escape}
	}

	return []key.Binding{km.Escape, km.Enter}
}

func (km keyMap) FullHelp() [][]key.Binding {
	if km.submitted {
		return [][]key.Binding{
			{km.Escape},
			{km.Help, km.Quit},
		}
	}

	return [][]key.Binding{
		{km.Escape, km.Enter},
	}
}

func newKeyMap(submitted bool) keyMap {
	return keyMap{
		submitted: submitted,
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
}
