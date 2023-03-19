package fileoptions

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db/models"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
)

const (
	selector = "→"
)

type FileOptions struct {
	cfg     *config.Config
	file    *models.File
	currIdx int
}

func New(cfg *config.Config) FileOptions {
	return FileOptions{
		cfg: cfg,
	}
}

func (m FileOptions) Init() tea.Cmd {
	return nil
}

func (m FileOptions) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		numOptions := len(m.visibleOptions())
		switch msg.String() {
		case "up", "k":
			m.currIdx = m.currIdx - 1
			if m.currIdx < 0 {
				m.currIdx = numOptions - 1
			}
		case "down", "j":
			m.currIdx = m.currIdx + 1
			if m.currIdx >= numOptions {
				m.currIdx = 0
			}
		case "enter":
			opt := m.visibleOptions()[m.currIdx]
			if opt == View {
				return m, cmds.PushView(views.Code)
			}

			var pk prompt.Kind
			switch opt {
			case Visiblity:
				pk = prompt.ChangeVisibility
			case Extension:
				pk = prompt.ChangeExtension
			case Sign:
				pk = prompt.GenerateSignedURL
			case Delete:
				pk = prompt.DeleteFile
			}

			return m, tea.Batch(
				cmds.PushView(views.Prompt),
				prompt.SetPromptKindCmd(pk),
			)
		}
	case msgs.PushView, msgs.PopView:
		// reset
		m.currIdx = 0
	case msgs.FileLoaded:
		m.file = msg.File
	case msgs.FileDeselected:
		m.file = nil
	}
	return m, cmd
}

func (m FileOptions) View() string {
	if m.file == nil {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Top, "", m.renderDetails(), m.renderOptions())
}

func (m FileOptions) renderDetails() string {
	str := strings.Builder{}

	httpAddr := m.cfg.HTTPAddressForFile(m.file.ID)
	visibility := "public"
	if m.file.Private {
		httpAddr = "<none> (requires a signed URL)"
		visibility = "private 🔒"
	}

	details := []struct {
		title  string
		values [][2]string
	}{
		{
			title: "details",
			values: [][2]string{
				{"id", m.file.ID},
				{"visibility", visibility},
				{"extension", m.file.Type},
				{"size", humanize.Bytes(m.file.Size)},
				{"created", fmt.Sprintf("%s (%s)", m.file.CreatedAt.Format(time.RFC3339), humanize.Time(m.file.CreatedAt))},
			},
		},
		{
			title: "access",
			values: [][2]string{
				{"web", httpAddr},
				{"ssh", m.cfg.SSHCommandForFile(m.file.ID)},
			},
		},
	}

	for _, detail := range details {
		str.WriteString(detail.title)
		str.WriteString(":\n")
		for _, value := range detail.values {
			str.WriteString("  ")
			str.WriteString(styles.C(styles.Colors.Blue, value[0]))
			str.WriteString(": ")
			str.WriteString(value[1])
			str.WriteRune('\n')
		}
	}

	return str.String()
}

func (m FileOptions) renderOptions() string {
	str := strings.Builder{}
	str.WriteString("options:\n")

	for i, opt := range m.visibleOptions() {
		isCurrentOption := i == m.currIdx
		color := styles.Colors.Yellow
		if opt == Delete {
			color = styles.Colors.Red
		}

		if isCurrentOption {
			str.WriteString(styles.C(color, selector))
		} else {
			str.WriteString(strings.Repeat(" ", lipgloss.Width(selector)))
		}
		str.WriteRune(' ')

		var option string
		switch opt {
		case View:
			option = "view"
		case Extension:
			option = "change extension"
		case Sign:
			option = "generate signed url"
		case Visiblity:
			if m.file.Private {
				option = "make public"
			} else {
				option = "make private"
			}
		case Delete:
			option = "delete file"
		}

		if isCurrentOption {
			str.WriteString(styles.C(color, option))
		} else {
			str.WriteString(option)
		}
		str.WriteRune('\n')
	}

	return str.String()
}

func (m FileOptions) visibleOptions() []Option {
	if m.file == nil {
		return []Option{}
	}

	opts := []Option{
		View,
	}

	// only allow changing extension of non-binary files
	if !m.file.IsBinary() {
		opts = append(opts, Extension)
	}

	// only allow signing of private files
	if m.file.Private {
		opts = append(opts, Sign)
	}

	opts = append(opts, Visiblity, Delete)

	return opts
}
