package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/dsn"
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/robherley/snips.sh/internal/web"
)

type App struct {
	SSH        *ssh.Service
	HTTP       *web.Service
	DB         *db.DB
	OnShutdown func(context.Context)
}

func (app *App) Boot() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	app.listen()

	sig := <-done
	slog.Warn("received signal, shutting down services", "signal", sig.String())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()

	app.shutdown(ctx)

	return nil
}

func (app *App) listen() {
	services := []interface {
		ListenAndServe() error
	}{
		app.SSH,
		app.HTTP,
	}

	for _, svc := range services {
		go func() {
			if err := svc.ListenAndServe(); err != nil {
				slog.Warn("service stopped", "err", err)
			}
		}()
	}
}

func (app *App) shutdown(ctx context.Context) {
	services := []interface {
		Shutdown(context.Context) error
	}{
		app.SSH,
		app.HTTP,
	}

	wg := sync.WaitGroup{}
	if app.OnShutdown != nil {
		wg.Go(func() {
			app.OnShutdown(ctx)
		})
	}

	for _, svc := range services {
		wg.Go(func() {
			if err := svc.Shutdown(ctx); err != nil {
				slog.Warn("shutdown error", "err", err)
			}
		})
	}

	wg.Wait()

	if err := app.DB.Close(); err != nil {
		slog.Warn("unable to close database", "err", err)
	}
}

func New(cfg *config.Config, assets web.Assets) (*App, error) {
	connection, err := dsn.Parse(cfg.DB.URL).NewDB(cfg)
	if err != nil {
		return nil, err
	}
	database := connection

	ssh, err := ssh.New(cfg, database)
	if err != nil {
		return nil, err
	}

	httpSvc, err := web.New(cfg, database, assets)
	if err != nil {
		return nil, err
	}

	return &App{
		SSH:  ssh,
		HTTP: httpSvc,
		DB:   database,
	}, nil
}
