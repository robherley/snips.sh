package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/rs/zerolog/log"
)

type App struct {
	SSH *ssh.Server
	DB  *db.DB
}

func (app *App) Start() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := app.SSH.ListenAndServe(); err != nil {
			log.Warn().Err(err)
		}
	}()

	sig := <-done
	log.Warn().Str("signal", sig.String()).Msg("received signal, shutting down app")
	if app.DB != nil {
		if err := app.DB.Close(); err != nil {
			log.Error().Err(err).Msg("unable to close db")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := app.SSH.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("unable to shutdown ssh server")
	}

	return nil
}

func New(cfg *config.Config) (*App, error) {
	db, err := db.New(cfg)
	if err != nil {
		return nil, err
	}

	ssh, err := ssh.New(cfg, db)
	if err != nil {
		return nil, err
	}

	return &App{
		SSH: ssh,
		DB:  db,
	}, nil
}
