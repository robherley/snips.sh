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

var (
	scanner     *magika.Scanner
	scannerOnce sync.Once
	scannerErr  error
)

// initScanner initializes the magika scanner once.
// The model and configuration files are embedded at build time.
func initScanner() (*magika.Scanner, error) {
	scannerOnce.Do(func() {
		start := time.Now()
		scanner, scannerErr = magika.NewScanner()
		if scannerErr != nil {
			slog.Error("failed to initialize magika scanner", "err", scannerErr)
		} else {
			slog.Info("magika scanner initialized", "dur", time.Since(start))
		}
	})
	return scanner, scannerErr
}

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
