package main

import (
	"embed"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/rs/zerolog/log"
)

var (
	//go:embed web/*
	webFS embed.FS
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg)

	snips, err := app.New(cfg, &webFS)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	err = snips.DB.AutoMigrate(db.AllModels...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to auto migrate db")
	}

	log.Info().Str("ssh_addr", cfg.SSHListenAddr()).Str("http_addr", cfg.HTTPListenAddr()).Msg("starting snips.sh")
	if err := snips.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
