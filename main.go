package main

import (
	"context"
	"embed"
	"flag"
	"log/slog"
	"os"

	"github.com/robherley/snips.sh/internal/app"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/stats"
	"github.com/robherley/snips.sh/internal/web"
)

var (
	//go:embed web/*
	webFS embed.FS
	//go:embed README.md
	readme []byte
	//go:embed docs/*.md
	docsFS embed.FS
)

func main() {
	logger.Initialize()

	cfg, err := config.Load()

	cfg.HTTP.Internal.Host = "0.0.0.0:8080"
	if err != nil {
		slog.Error("unable to load config", "err", err)
		os.Exit(1)
	}

	if cfg.Debug {
		logger.Initialize(slog.LevelDebug)
	}

	statsd, err := stats.Initialize(cfg.Metrics.Statsd, cfg.Metrics.UseDogStatsd)
	if err != nil {
		slog.Error("unable to initialize metrics", "err", err)
		os.Exit(1)
	}

	usage := flag.Bool("usage", false, "print environment variable usage")
	flag.Parse()
	if usage != nil && *usage {
		_ = cfg.PrintUsage()
		return
	}

	assets, err := web.NewAssets(
		&webFS,
		&docsFS,
		readme,
		cfg.HTML.ExtendHeadFile,
	)
	if err != nil {
		slog.Error("failed to load assets", "err", err)
		os.Exit(1)
	}

	application, err := app.New(cfg, assets)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	application.OnShutdown = func(_ context.Context) {
		statsd.Shutdown()
	}

	if err := application.DB.Migrate(context.Background()); err != nil {
		slog.Error("failed to migrate database", "err", err)
		os.Exit(1)
	}

	slog.Info("starting snips.sh",
		"ssh_addr", cfg.SSH.Internal.String(),
		"http_addr", cfg.HTTP.Internal.String(),
	)

	if err := application.Boot(); err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}
}
