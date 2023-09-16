package prompt

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	submitted bool

	Help   key.Binding
	Quit   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	bindings := []key.Binding{km.Help, km.Escape}

	if !km.submitted {
		bindings = append(bindings, km.Enter)
	}

	return bindings
}

func (km keyMap) FullHelp() [][]key.Binding {
	bindings := [][]key.Binding{
		{km.Escape},
		{km.Help, km.Quit},
	}

	if !km.submitted {
		bindings[0] = append(bindings[0], km.Enter)
	}

	return bindings
}

func newKeyMap(submitted bool) keyMap {
	km := keyMap{
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

	km.submitted = submitted
	return km
}
