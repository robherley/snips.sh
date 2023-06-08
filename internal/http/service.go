package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database db.DB, assets *Assets) (*Service, error) {
	router := chi.NewRouter()

	router.Use(WithRequestID)
	router.Use(WithLogger)
	router.Use(WithMetrics)
	router.Use(WithRecover)

	router.Get("/", DocHandler("README.md", assets))
	router.Get("/docs/", func(w http.ResponseWriter, r *http.Request) {
		DocHandler(r.URL.Path[len("/docs/"):], assets)(w, r)
	})
	router.Get("/health", HealthHandler)
	router.Get("/f/{fileID}", FileHandler(cfg, database, assets.Template()))
	router.Get("/assets/index.js", assets.ServeJS)
	router.Get("/assets/index.css", assets.ServeCSS)
	router.Get("/meta.json", MetaHandler(cfg))

	if cfg.Debug {
		router.Mount("/_debug", middleware.Profiler())
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTP.Internal.Host,
		Handler: router,
	}

	return &Service{httpServer}, nil
}
