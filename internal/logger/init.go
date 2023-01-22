package logger

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(cfg *config.Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}
