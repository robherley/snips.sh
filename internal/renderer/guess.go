//go:build cgo && onnxruntime && !noguesser

package renderer

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/google/magika/go/magika"
)

var (
	scanner     *magika.Scanner
	scannerOnce sync.Once
	scannerErr  error
)

// initScanner initializes the magika scanner once.
// It reads configuration from environment variables:
//   - MAGIKA_ASSETS_DIR: path to magika assets directory (default: /opt/magika/assets)
//   - MAGIKA_MODEL: model name to use (default: standard_v3_3)
func initScanner() (*magika.Scanner, error) {
	scannerOnce.Do(func() {
		// TODO(robherley): cleanup
		assetsDir := os.Getenv("MAGIKA_ASSETS_DIR")
		if assetsDir == "" {
			assetsDir = "/opt/magika/assets"
		}

		// TODO(robherley): cleanup
		modelName := os.Getenv("MAGIKA_MODEL")
		if modelName == "" {
			modelName = "standard_v3_3"
		}

		scanner, scannerErr = magika.NewScanner(assetsDir, modelName)
		if scannerErr != nil {
			slog.Error("failed to initialize magika scanner", "err", scannerErr, "assetsDir", assetsDir, "model", modelName)
		} else {
			slog.Info("magika scanner initialized", "assetsDir", assetsDir, "model", modelName)
		}
	})
	return scanner, scannerErr
}

// contentReader implements io.ReaderAt for a string.
type contentReader struct {
	content string
}

func (r *contentReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(r.content)) {
		return 0, nil
	}
	n = copy(p, r.content[off:])
	return n, nil
}

func Guess(content string) string {
	guessStart := time.Now()
	defer metrics.MeasureSince([]string{"guess", "duration"}, guessStart)

	s, err := initScanner()
	if err != nil || s == nil {
		slog.Warn("magika scanner not available", "err", err)
		return ""
	}

	reader := &contentReader{content: content}
	ct, err := s.Scan(reader, len(content))
	if err != nil {
		slog.Warn("failed to scan content with magika", "err", err)
		return ""
	}

	return strings.ToLower(ct.Label)
}
