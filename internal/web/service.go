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

	mux.HandleFunc("GET /{$}", DocHandler(cfg, assets))
	mux.HandleFunc("GET /docs/{name}", DocHandler(cfg, assets))
	mux.HandleFunc("GET /og.png", DocOGImageHandler(cfg, assets))
	mux.HandleFunc("GET /docs/{name}/og.png", DocOGImageHandler(cfg, assets))
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("GET /f/{fileID}", WithFile(cfg, database, FileHandler(cfg, database, assets)))
	mux.HandleFunc("GET /f/{fileID}/rev", WithFile(cfg, database, RevisionsHandler(cfg, database, assets)))
	mux.HandleFunc("GET /f/{fileID}/rev/{revisionID}", WithFile(cfg, database, RevisionDiffHandler(database, assets)))
	mux.HandleFunc("GET /f/{fileID}/og.png", WithFile(cfg, database, OGImageHandler(cfg, assets)))
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
