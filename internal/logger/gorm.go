package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	gormLogger "gorm.io/gorm/logger"
)

type GormAdapter struct {
	ZL zerolog.Logger
}

func (ga *GormAdapter) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return ga
}

func (ga *GormAdapter) Error(_ context.Context, msg string, opts ...interface{}) {
	ga.ZL.Error().Msg(fmt.Sprintf(msg, opts...))
}

func (ga *GormAdapter) Warn(_ context.Context, msg string, opts ...interface{}) {
	ga.ZL.Warn().Msg(fmt.Sprintf(msg, opts...))
}

func (ga *GormAdapter) Info(_ context.Context, msg string, opts ...interface{}) {
	ga.ZL.Info().Msg(fmt.Sprintf(msg, opts...))
}

func (ga *GormAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	ga.ZL.Trace().Str("sql", sql).Int64("rows", rows).Dur("elapsed", time.Since(begin)).Msg("gorm")
}
