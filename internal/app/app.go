package app

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/http"
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/rs/zerolog/log"
)

type App struct {
	SSH  *ssh.Service
	HTTP *http.Service
	DB   *db.DB
}

type service interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

func (app *App) Start() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	services := []service{
		app.SSH,
		app.HTTP,
	}

	start(services)

	sig := <-done
	log.Warn().Str("signal", sig.String()).Msg("received signal, shutting down services")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()

	stop(ctx, services)

	return nil
}

func start(services []service) {
	for i := range services {
		go func(svc service) {
			if err := svc.ListenAndServe(); err != nil {
				log.Warn().Err(err)
			}
		}(services[i])
	}
}

func stop(ctx context.Context, services []service) {
	wg := sync.WaitGroup{}
	wg.Add(len(services))

	for i := range services {
		go func(svc service) {
			defer wg.Done()
			if err := svc.Shutdown(ctx); err != nil {
				log.Warn().Err(err)
			}
		}(services[i])
	}
}

func New(cfg *config.Config, webFS *embed.FS, readme string) (*App, error) {
	db, err := db.New(cfg)
	if err != nil {
		return nil, err
	}

	ssh, err := ssh.New(cfg, db)
	if err != nil {
		return nil, err
	}

	http, err := http.New(cfg, db, webFS, readme)
	if err != nil {
		return nil, err
	}

	return &App{
		SSH:  ssh,
		HTTP: http,
		DB:   db,
	}, nil
}
