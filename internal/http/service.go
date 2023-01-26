package http

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, db *db.DB, webFS *embed.FS) (*Service, error) {
	mux := http.NewServeMux()

	templates := template.Must(template.ParseFS(webFS, "web/templates/*"))

	// TODO(robherley): gzip?
	static, err := fs.Sub(webFS, "web/static")
	if err != nil {
		return nil, err
	}

	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/f/", FileHandler(cfg, db, templates))
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.FS(static))))

	httpServer := &http.Server{
		Addr:    cfg.HTTPListenAddr(),
		Handler: WithMiddleware(mux, WithRecover, WithLogger, WithRequestID),
	}

	return &Service{httpServer}, nil
}
