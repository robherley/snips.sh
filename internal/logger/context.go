package logger

import (
	"context"
	"log/slog"
)

type ctxkey string

var ContextKey ctxkey = "logger"

func From(ctx context.Context) *slog.Logger {
	ctxLogger := ctx.Value(ContextKey)
	if ctxLogger == nil {
		return slog.Default()
	}
	return ctx.Value(ContextKey).(*slog.Logger)
}
