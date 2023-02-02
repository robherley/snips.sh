package logger

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ctxkey string

var ContextKey ctxkey = "logger"

func From(ctx context.Context) *zerolog.Logger {
	ctxLogger := ctx.Value(ContextKey)
	if ctxLogger == nil {
		return &log.Logger
	}
	return ctx.Value(ContextKey).(*zerolog.Logger)
}
