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
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/robherley/snips.sh/internal/web"
)

type App struct {
	SSH        *ssh.Service
	HTTP       *web.Service
	DB         db.DB
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
	type listenable interface {
		ListenAndServe() error
	}

	services := []listenable{
		app.SSH,
		app.HTTP,
	}

	for i := range services {
		go func(svc listenable) {
			if err := svc.ListenAndServe(); err != nil {
				slog.Warn("service stopped", "err", err)
			}
		}(services[i])
	}
}

func (app *App) shutdown(ctx context.Context) {
	type shutdownable interface {
		Shutdown(context.Context) error
	}

	services := []shutdownable{
		app.SSH,
		app.HTTP,
	}

	wg := sync.WaitGroup{}
	wg.Add(len(services))

	if app.OnShutdown != nil {
		wg.Add(1)
		go func(a *App) {
			defer wg.Done()
			a.OnShutdown(ctx)
		}(app)
	}

	for i := range services {
		go func(svc shutdownable) {
			defer wg.Done()
			if err := svc.Shutdown(ctx); err != nil {
				slog.Warn("shutdown error", "err", err)
			}
		}(services[i])
	}

	wg.Wait()
}

func New(cfg *config.Config, assets web.Assets) (*App, error) {
	database, err := db.NewSqlite(cfg.DB.FilePath)
	if err != nil {
		return nil, err
	}

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
