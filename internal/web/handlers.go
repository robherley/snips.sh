package web

import (
	"net/http"
	"net/http/pprof"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("💚\n"))
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	profiler := r.PathValue("profile")
	switch profiler {
	case "cmdline":
		pprof.Cmdline(w, r)
	case "profile":
		pprof.Profile(w, r)
	case "symbol":
		pprof.Symbol(w, r)
	case "trace":
		pprof.Trace(w, r)
	default:
		// Available profiles can be found in [runtime/pprof.Profile].
		// https://pkg.go.dev/runtime/pprof#Profile
		pprof.Handler(profiler).ServeHTTP(w, r)
	}
}
