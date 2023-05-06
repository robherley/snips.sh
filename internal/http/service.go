package http

import (
	"embed"
	"net/http"
	"net/http/pprof"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database db.DB, webFS *embed.FS, readme string) (*Service, error) {
	assets, err := NewAssets(webFS)
	if err != nil {
		return nil, err
	}

	r := NewRouter()
	r.HandleFunc("/", IndexHandler(readme, assets.Templates()))
	r.HandleFunc("/health", HealthHandler)
	r.HandleFunc("/f/", FileHandler(cfg, database, assets.Templates()))
	r.HandleFunc("/assets/index.js", assets.ServeJS)
	r.HandleFunc("/assets/index.css", assets.ServeCSS)

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
