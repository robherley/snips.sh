package web

import (
	"net/http"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database db.DB, assets Assets) (*Service, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", DocHandler(assets))
	mux.HandleFunc("GET /docs/{name}", DocHandler(assets))
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("GET /f/{fileID}", FileHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/og.png", OGImageHandler(cfg, database, assets))
	mux.HandleFunc("GET /assets/{asset...}", assets.Serve)
	mux.HandleFunc("GET /meta.json", MetaHandler(cfg))

	if cfg.Debug {
		mux.HandleFunc("/_debug/pprof/{profile}", ProfileHandler)
	}

	return &Service{
		&http.Server{
			Addr:    cfg.HTTP.Internal.Host,
			Handler: WithMiddleware(mux),
		},
	}, nil
}
