package prompt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/timeutil"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// signedURLDialog generates a signed URL for a private file, valid for a
// user-supplied duration.
type signedURLDialog struct {
	textDialog
	step int
	ttl  time.Duration
}

func newSignedURLDialog() *signedURLDialog {
	return &signedURLDialog{textDialog: newTextDialog()}
}

func (d *signedURLDialog) title() string {
	return "generate signed url"
}

func (d *signedURLDialog) question(file *snips.File) string {
	if d.step == 0 {
		return fmt.Sprintf("How long do you want the signed url for %q to last for?\n%s",
			file.ID, styles.C(styles.Colors.Muted, "(e.g. 30s, 5m, 3h)"))
	}

	return fmt.Sprintf("Should the signed url for %q burn after read?\n(y/n)", file.ID)
}

func (d *signedURLDialog) submit(e env) tea.Cmd {
	if d.step == 0 {
		dur, err := timeutil.ParseDuration(d.value())
		if err != nil {
			return SetPromptErrorCmd(err)
		}

		if dur <= 0 {
			return SetPromptErrorCmd(errors.New("duration must be greater than 0"))
		}

		d.ttl = dur
		d.step = 1
		d.input.SetValue("")
		return nil
	}

	answer := strings.ToLower(strings.TrimSpace(d.value()))
	if answer == "" {
		return SetPromptErrorCmd(errors.New("please specify yes or no"))
	}

	burnAfterRead := false
	switch answer[0] {
	case 'y':
		burnAfterRead = true
	case 'n':
	default:
		return SetPromptErrorCmd(errors.New("please specify yes or no"))
	}

	url, expires := e.file.GetSignedURL(e.cfg, d.ttl, burnAfterRead)

	metrics.IncrCounter([]string{"file", "sign"}, 1)
	logger.From(e.ctx).Info("private file signed", "file_id", e.file.ID, "expires_at", expires, "burn_after_read", burnAfterRead)

	// keep the url on a single unwrapped line and hyperlink it, so it stays
	// easy to copy (or cmd+click) out of the modal
	raw := url.String()
	link := lipgloss.NewStyle().Hyperlink(raw).Render(raw)
	msg := "expires at: " + expires.Format(time.RFC3339)
	if burnAfterRead {
		msg += "\n\nburn after read: yes"
	}
	return SetPromptFeedbackCmd(feedback.Success(link+"\n\n"+msg), true)
}
