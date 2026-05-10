package views

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
)

type Kind int

const (
	Browser Kind = iota
	Code
	Prompt
	Settings
)

type Model interface {
	tea.Model
	Keys() help.KeyMap
	// IsCapturing reports whether the view is currently consuming raw keystrokes
	// (e.g. typing into a filter or text input) and therefore the TUI shouldn't
	// intercept its own shortcuts.
	IsCapturing() bool
}
