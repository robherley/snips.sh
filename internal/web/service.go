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

	mux.HandleFunc("GET /{$}", LandingHandler(cfg, assets))
	mux.HandleFunc("GET /docs/{name}", DocHandler(cfg, assets))
	mux.HandleFunc("GET /og.png", DocOGImageHandler(cfg, assets))
	mux.HandleFunc("GET /docs/{name}/og.png", DocOGImageHandler(cfg, assets))
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("GET /f/{fileID}", FileHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/rev", RevisionsHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/rev/{revisionID}", RevisionDiffHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/og.png", OGImageHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/n/{name}", FileHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/n/{name}/rev", RevisionsHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/n/{name}/rev/{revisionID}", RevisionDiffHandler(cfg, database, assets))
	mux.HandleFunc("GET /f/{fileID}/n/{name}/og.png", OGImageHandler(cfg, database, assets))
	mux.HandleFunc("GET /assets/{asset...}", assets.Serve)

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
