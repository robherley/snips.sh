package web

import (
	"net/http"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database *db.DB, assets Assets) (*Service, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	NewUI(cfg, database, assets).Register(mux)
	NewAPI(cfg, database).Register(mux)

	if cfg.Debug {
		mux.HandleFunc("/_debug/pprof/{profile}", WithLocalhostOnly(ProfileHandler))
	}

	return &Service{
		&http.Server{
			Addr:    cfg.HTTP.Internal.Host,
			Handler: WithMiddleware(mux),
		},
	}, nil
}
