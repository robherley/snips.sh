package http

import (
	"context"
	"net/http"
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/rs/zerolog/log"
)

type Middleware func(next http.Handler) http.Handler

// WithMiddleware is a helper function to apply multiple middlewares to a handler.
func WithMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	withMiddleware := handler
	for i := range middlewares {
		withMiddleware = middlewares[i](withMiddleware)
	}
	return withMiddleware
}

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
