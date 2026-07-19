package prompt

import (
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// signedURLDialog generates a signed URL for a private file, valid for a
// user-supplied duration.
type signedURLDialog struct {
	textDialog
}

func newSignedURLDialog() *signedURLDialog {
	return &signedURLDialog{newTextDialog()}
}

func (d *signedURLDialog) title() string {
	return "generate signed url"
}

func (d *signedURLDialog) question(file *snips.File) string {
	return fmt.Sprintf("How long do you want the signed url for %q to last for?\n%s",
		file.ID, styles.C(styles.Colors.Muted, "(e.g. 30s, 5m, 3h)"))
}

func (d *signedURLDialog) submit(e env) tea.Cmd {
	dur, err := time.ParseDuration(d.value())
	if err != nil {
		return SetPromptErrorCmd(err)
	}

	if dur <= 0 {
		return SetPromptErrorCmd(errors.New("duration must be greater than 0"))
	}

	url, expires := e.file.GetSignedURL(e.cfg, dur)

	metrics.IncrCounter([]string{"file", "sign"}, 1)
	logger.From(e.ctx).Info("private file signed", "file_id", e.file.ID, "expires_at", expires)

	// keep the url on a single unwrapped line and hyperlink it, so it stays
	// easy to copy (or cmd+click) out of the modal
	raw := url.String()
	link := lipgloss.NewStyle().Hyperlink(raw).Render(raw)
	msg := feedback.Success(link + "\n\n" + "expires at: " + expires.Format(time.RFC3339))
	return SetPromptFeedbackCmd(msg, true)
}
