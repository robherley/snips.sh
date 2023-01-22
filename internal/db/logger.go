package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	gormLogger "gorm.io/gorm/logger"
)

type logger struct {
	zl zerolog.Logger
}

func (log *logger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return log
}

func (log *logger) Error(_ context.Context, msg string, opts ...interface{}) {
	log.zl.Error().Msg(fmt.Sprintf(msg, opts...))
}

func (log *logger) Warn(_ context.Context, msg string, opts ...interface{}) {
	log.zl.Warn().Msg(fmt.Sprintf(msg, opts...))
}

func (log *logger) Info(_ context.Context, msg string, opts ...interface{}) {
	log.zl.Info().Msg(fmt.Sprintf(msg, opts...))
}

func (log *logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	log.zl.Trace().Str("sql", sql).Int64("rows", rows).Dur("elapsed", time.Since(begin)).Msg("gorm")
}
