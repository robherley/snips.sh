package http

import (
	"net/http"
	"time"

	"github.com/armon/go-metrics"
)

type Router struct {
	*http.ServeMux
}

func NewRouter() *Router {
	return &Router{http.NewServeMux()}
}

func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if handler == nil {
		panic("router: nil handler")
	}
	r.Handle(pattern, http.HandlerFunc(handler))
}

func (r *Router) Handle(pattern string, handler http.Handler) {
	r.ServeMux.Handle(pattern, recordMetrics(pattern, handler))
}

func recordMetrics(pattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		labels := []metrics.Label{
			{Name: "path", Value: pattern},
			{Name: "method", Value: r.Method},
		}

		start := time.Now()
		metrics.IncrCounterWithLabels([]string{"http", "request"}, 1, labels)
		next.ServeHTTP(w, r)
		metrics.MeasureSinceWithLabels([]string{"http", "request", "duration"}, start, labels)
	})
}
