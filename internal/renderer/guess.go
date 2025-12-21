//go:build amd64 && !noguesser

package renderer

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/guesslang-go/pkg/guesser"
)

var guesslang *guesser.Guesser

func init() {
	var err error
	guesslang, err = guesser.New()
	if err != nil {
		slog.Error("failed to initialize guesslang", "err", err)
		os.Exit(1)
	}
}

func Guess(content string) string {
	guessStart := time.Now()
	answer, err := guesslang.Guess(content)
	metrics.MeasureSince([]string{"guess", "duration"}, guessStart)
	if err != nil {
		slog.Warn("failed to guess the file type", "err", err)
		return ""
	}

	return strings.ToLower(answer.Predictions[0].Language)
}
