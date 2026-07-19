// Package feedback renders success/error notices shown inside views.
package feedback

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Feedback struct {
	Text string
	Err  bool
}

func Success(text string) Feedback { return Feedback{Text: text} }
func Error(text string) Feedback   { return Feedback{Text: text, Err: true} }

func (f Feedback) Empty() bool {
	return f.Text == ""
}

func (f Feedback) View() string {
	if f.Empty() {
		return ""
	}

	icon := styles.C(styles.Colors.Green, "✓")
	if f.Err {
		icon = styles.C(styles.Colors.Red, "✗")
	}

	text := lipgloss.NewStyle().Foreground(styles.Colors.White).Render(f.Text)
	return icon + " " + strings.ReplaceAll(text, "\n", "\n  ")
}
