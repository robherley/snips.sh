package ssh

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Confirm struct {
	Question string
}

func (c *Confirm) Questionf(format string, v ...interface{}) {
	c.Question = fmt.Sprintf(format, v...)
}

func (c *Confirm) Prompt(sesh ssh.Session) (bool, error) {
	style := lipgloss.NewStyle().Foreground(styles.Colors.Yellow)
	prompt := style.Render(c.Question + " [y/N] ")
	wish.Print(sesh, prompt)

	option := make([]byte, 1)
	n, err := sesh.Read(option)
	if err != nil {
		return false, err
	}

	if n == 0 {
		return false, nil
	}

	if option[0] == 'y' || option[0] == 'Y' {
		return true, nil
	}

	return false, nil
}
