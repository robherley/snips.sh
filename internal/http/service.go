package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	api_v1 "github.com/robherley/snips.sh/internal/http/api/v1"
)

type Service struct {
	*http.Server
	Router *chi.Mux
}

func New(cfg *config.Config, database db.DB, assets Assets) (*Service, error) {
	router := chi.NewRouter()

	router.Use(WithRequestID)
	router.Use(WithLogger)
	router.Use(WithMetrics)
	router.Use(WithRecover)

	router.Get("/", DocHandler(assets))
	router.Get("/docs/{name}", DocHandler(assets))
	router.Get("/health", HealthHandler)
	router.Get("/f/{fileID}", FileHandler(cfg, database, assets))
	router.Get("/assets/index.js", assets.ServeJS)
	router.Get("/assets/index.css", assets.ServeCSS)
	router.Get("/meta.json", MetaHandler(cfg))

	if cfg.EnableAPI {
		router.Mount("/api/v1", api_v1.ApiHandler(cfg, database))
	}

	if cfg.Debug {
		router.Mount("/_debug", middleware.Profiler())
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTP.Internal.Host,
		Handler: router,
	}

	return &Service{httpServer, router}, nil
}
