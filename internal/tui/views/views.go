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
)

type Model interface {
	tea.Model
	Keys() help.KeyMap
}
