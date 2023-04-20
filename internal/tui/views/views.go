package views

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
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
