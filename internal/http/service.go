package http

import (
	"net/http"
	"net/http/pprof"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
)

type Service struct {
	*http.Server
}

func New(cfg *config.Config, database db.DB, assets *Assets) (*Service, error) {
	r := NewRouter()
	r.HandleFunc("/", DocHandler("README.md", assets))
	r.HandleFunc("/docs/", func(w http.ResponseWriter, r *http.Request) {
		DocHandler(r.URL.Path[len("/docs/"):], assets)(w, r)
	})
	r.HandleFunc("/health", HealthHandler)
	r.HandleFunc("/f/", FileHandler(cfg, database, assets.Template()))
	r.HandleFunc("/assets/index.js", assets.ServeJS)
	r.HandleFunc("/assets/index.css", assets.ServeCSS)
	r.HandleFunc("/meta.json", MetaHandler(cfg))

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
