package http

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/pprof"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, db *db.DB, webFS *embed.FS, readme string) (*Service, error) {
	mux := http.NewServeMux()

	templates := template.Must(template.ParseFS(webFS, "web/templates/*"))

	// TODO(robherley): gzip?
	static, err := fs.Sub(webFS, "web/static")
	if err != nil {
		return nil, err
	}

	mux.HandleFunc("/", IndexHandler(readme, templates))
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/f/", FileHandler(cfg, db, templates))
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.FS(static))))

	if cfg.Debug {
		mux.HandleFunc("/_debug/profile", pprof.Profile)
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTP.Internal.Host,
		Handler: WithMiddleware(mux, WithRecover, WithLogger, WithRequestID),
	}

	return &Service{httpServer}, nil
}
