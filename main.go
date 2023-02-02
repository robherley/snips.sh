package main

import (
	"embed"
	"flag"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/rs/zerolog/log"
)

var (
	//go:embed web/*
	webFS embed.FS
	//go:embed README.md
	readme string
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg)

	usage := flag.Bool("usage", false, "print environment variable usage")
	flag.Parse()
	if usage != nil && *usage {
		_ = cfg.PrintUsage()
		return
	}

	snips, err := app.New(cfg, &webFS, readme)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	err = snips.DB.AutoMigrate(db.AllModels...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to auto migrate db")
	}

	log.Info().Str("ssh_addr", cfg.SSH.Internal.String()).Str("http_addr", cfg.HTTP.Internal.String()).Msg("starting snips.sh")
	if err := snips.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
