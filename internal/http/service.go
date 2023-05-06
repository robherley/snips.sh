package http

import (
	"embed"
	"html/template"
	"net/http"
	"net/http/pprof"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database db.DB, webFS *embed.FS, readme string) (*Service, error) {
	templates := template.Must(template.ParseFS(webFS, "web/templates/*"))

	assets, err := MinififyStaticAssets(webFS)
	if err != nil {
		return nil, err
	}

	r := NewRouter()
	r.HandleFunc("/", IndexHandler(readme, templates))
	r.HandleFunc("/health", HealthHandler)
	r.HandleFunc("/f/", FileHandler(cfg, database, templates))
	r.HandleFunc("/assets/index.js", assets.Handler)
	r.HandleFunc("/assets/index.css", assets.Handler)

	if cfg.Debug {
		r.HandleFunc("/_debug/pprof/", pprof.Index)
		r.HandleFunc("/_debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/_debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/_debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/_debug/pprof/trace", pprof.Trace)
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTP.Internal.Host,
		Handler: WithMiddleware(r, WithRecover, WithLogger, WithRequestID),
	}

	return &Service{httpServer}, nil
}
