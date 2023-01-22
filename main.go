package main

import (
	"github.com/rs/zerolog/log"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg)

	snips, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	err = snips.DB.AutoMigrate(db.AllModels...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to auto migrate db")
	}

	log.Info().Str("ssh_address", cfg.SSHAddress()).Msg("starting snips.sh")
	if err := snips.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
