package http

import (
	"context"
	"net/http"
	"time"

	"github.com/armon/go-metrics"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
)

// WithRequestID adds a unique request ID to the request context.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := id.New()
		r.Header.Set(RequestIDHeader, requestID)

		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithLogger adds a request scoped logger to the request context.
func WithLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := r.RemoteAddr
		start := time.Now()
		requestID := r.Header.Get(RequestIDHeader)

		reqLog := log.With().Fields(map[string]interface{}{
			"svc":        "http",
			"addr":       addr,
			"request_id": requestID,
			"path":       r.URL.Path,
		}).Logger()

		ctx := context.WithValue(r.Context(), logger.ContextKey, &reqLog)
		reqLog.Info().Msg("connected")

		next.ServeHTTP(w, r.WithContext(ctx))

		reqLog.Info().Dur("dur", time.Since(start)).Msg("disconnected")
	})
}

// WithRecover will recover from any panics and log them.
func WithRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.From(r.Context()).Error().Msgf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// WithMetrics will record metrics for the request.
func WithMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		rctx := chi.RouteContext(r.Context())
		pattern := rctx.RoutePattern()
		if pattern == "" {
			// empty pattern, didn't match router e.g. 404
			pattern = "*"
		}

		labels := []metrics.Label{
			{Name: "path", Value: pattern},
			{Name: "method", Value: r.Method},
		}

		metrics.IncrCounterWithLabels([]string{"http", "request"}, 1, labels)
		metrics.MeasureSinceWithLabels([]string{"http", "request", "duration"}, start, labels)
	})
}
