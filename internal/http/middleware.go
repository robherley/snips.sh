package http

import (
	"context"
	"net/http"
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/rs/zerolog/log"
)

type Middleware func(next http.Handler) http.Handler

func WithMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	withMiddleware := handler
	for i := range middlewares {
		withMiddleware = middlewares[i](withMiddleware)
	}
	return withMiddleware
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := id.MustGenerate()
		r.Header.Set(RequestIDHeader, requestID)

		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := r.RemoteAddr
		start := time.Now()
		requestID := r.Header.Get(RequestIDHeader)

		reqLog := log.With().Str("svc", "http").Str("addr", addr).Str("request_id", requestID).Logger()

		ctx := context.WithValue(r.Context(), LoggerContextKey, reqLog)
		reqLog.Info().Msg("connected")
		next.ServeHTTP(w, r.WithContext(ctx))
		reqLog.Info().Dur("dur", time.Since(start)).Msg("disconnected")
	})
}
