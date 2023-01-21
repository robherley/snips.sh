package main

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
)

func init() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	snips, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	log.Info().Str("ssh_address", cfg.SSHAddress()).Msg("starting snips.sh")
	if err := snips.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
