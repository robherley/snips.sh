package settings

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// deleteView is the delete-all-my-data page: a typed confirmation before
// permanently removing every file.
type deleteView struct {
	deps

	confirm   textinput.Model // typed confirmation of the user's id
	fileCount int64
	feedback  feedback.Feedback
}

func newDeleteView(d deps) deleteView {
	ti := textinput.New()
	ti.CharLimit = 255
	ti.SetWidth(30)
	ti.Prompt = styles.BC(styles.Colors.Red, "> ")

	return deleteView{
		deps:    d,
		confirm: ti,
	}
}

// enter counts the files at stake and focuses the confirmation input.
func (m deleteView) enter() (deleteView, tea.Cmd, error) {
	count, err := m.db.CountFilesByUser(m.ctx, m.user.ID)
	if err != nil {
		return m, nil, fmt.Errorf("failed to count files: %w", err)
	}

	m.fileCount = count
	m.feedback = feedback.Feedback{}
	m.confirm.Reset()
	return m, m.confirm.Focus(), nil
}

func (m deleteView) update(msg tea.KeyPressMsg) (deleteView, result) {
	switch msg.String() {
	case "enter":
		if m.confirm.Value() != m.user.ID {
			m.feedback = feedback.Error("please type your user id to confirm")
			return m, result{}
		}
		return m.deleteEverything()
	case "esc":
		return m, result{back: true}
	}

	// everything else is typed into the confirmation input
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, result{cmd: cmd}
}

func (m deleteView) deleteEverything() (deleteView, result) {
	count, err := m.db.DeleteFilesByUser(m.ctx, m.user.ID)
	if err != nil {
		m.feedback = feedback.Error("failed to delete: " + err.Error())
		return m, result{}
	}

	metrics.IncrCounter([]string{"file", "delete", "all"}, 1)
	logger.From(m.ctx).Info("deleted all user files", "user_id", m.user.ID, "count", count)

	deleted := fmt.Sprintf("deleted %d files", count)
	if count == 1 {
		deleted = "deleted 1 file"
	}

	return m, result{
		back: true,
		fb:   feedback.Success(deleted),
		cmd:  cmds.ReloadFiles(m.db, m.user.ID),
	}
}

// rows renders the delete confirmation page.
func (m deleteView) rows() []string {
	warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Red)
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)

	things := fmt.Sprintf("all %d of your files", m.fileCount)
	switch m.fileCount {
	case 0:
		things = "your files (you have none)"
	case 1:
		things = "your only file"
	}

	rows := []string{
		warnStyle.Render(fmt.Sprintf("this permanently deletes %s.", things)),
		mutedStyle.Render("type your user id (" + m.user.ID + ") to confirm"),
		"",
		m.confirm.View(),
	}

	if !m.feedback.Empty() && m.feedback.Err {
		rows = append(rows, "", m.feedback.View())
	}

	return rows
}

// deleteKeyMap is shown while the delete confirmation input is focused; every
// other key is typed into the input.
type deleteKeyMap struct {
	Enter key.Binding
	Esc   key.Binding
	Quit  key.Binding
}

func (k deleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Esc, k.Quit}
}

func (k deleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Esc, k.Quit}}
}

var deleteKeys = deleteKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "confirm"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}
