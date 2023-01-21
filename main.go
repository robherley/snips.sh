package main

import (
	"github.com/rs/zerolog/log"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/util"
)

func init() {
	util.InitLogger()
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

	sqlv, err := snips.DB.Version()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to get sqlite version")
	}
	log.Info().Str("file", cfg.DB.FilePath).Str("version", sqlv).Msg("sqlite connected")

	log.Info().Str("ssh_address", cfg.SSHAddress()).Msg("starting snips.sh")
	if err := snips.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
