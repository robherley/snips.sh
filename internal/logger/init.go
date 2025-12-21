package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const DefaultLevel = slog.LevelInfo

func Initialize(lvls ...slog.Level) {
	lvl := DefaultLevel
	if len(lvls) > 0 {
		lvl = lvls[len(lvls)-1]
	}

	var handler slog.Handler

	if isatty.IsTerminal(os.Stderr.Fd()) {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      lvl,
			TimeFormat: time.TimeOnly,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		})
	}

	slog.SetDefault(slog.New(handler))
}
