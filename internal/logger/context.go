package logger

import (
	"context"

	"github.com/rs/zerolog"
)

type ctxkey string

var ContextKey ctxkey = "logger"

func From(ctx context.Context) *zerolog.Logger {
	return ctx.Value(ContextKey).(*zerolog.Logger)
}
