package main

import (
	"context"
	"embed"
	"flag"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/stats"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	//go:embed web/*
	webFS embed.FS
	//go:embed README.md
	readme string
)

func main() {
	logger.Initialize()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load config")
	}

	metrics, err := stats.Initialize(cfg.Metrics.Statsd)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to initialize metrics")
	}

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	usage := flag.Bool("usage", false, "print environment variable usage")
	flag.Parse()
	if usage != nil && *usage {
		_ = cfg.PrintUsage()
		return
	}

	application, err := app.New(cfg, &webFS, readme)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	application.OnShutdown = func(ctx context.Context) {
		metrics.Shutdown()
	}

	if err := application.DB.Migrate(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate database")
	}

	log.Info().Str("ssh_addr", cfg.SSH.Internal.String()).Str("http_addr", cfg.HTTP.Internal.String()).Msg("starting snips.sh")
	if err := application.Boot(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
}
