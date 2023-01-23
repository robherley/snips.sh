package http

import (
	"net/http"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, db *db.DB) (*Service, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/f/", FileHandler(db))
	// TODO(robherley): cute landing page

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddress(),
		Handler: mux,
	}

	return &Service{httpServer}, nil
}
