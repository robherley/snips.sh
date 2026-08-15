//go:build !noguesser

package renderer

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/magika-go/pkg/magika"
)

// initScanner initializes the magika scanner once.
// The model and configuration files are embedded at build time.
var initScanner = sync.OnceValues(func() (*magika.Scanner, error) {
	start := time.Now()
	scanner, err := magika.NewScanner()
	if err != nil {
		slog.Error("failed to initialize magika scanner", "err", err)
	} else {
		slog.Info("magika scanner initialized", "dur", time.Since(start))
	}
	return scanner, err
})

func Guess(content string) string {
	guessStart := time.Now()
	defer metrics.MeasureSince([]string{"guess", "duration"}, guessStart)

	s, err := initScanner()
	if err != nil || s == nil {
		slog.Warn("magika scanner not available", "err", err)
		return ""
	}

	ct, err := s.ScanString(content)
	if err != nil {
		slog.Warn("failed to scan content with magika", "err", err)
		return ""
	}

	return strings.ToLower(ct.Label)
}
