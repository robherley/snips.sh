package web

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
)

type Middleware func(next http.Handler) http.Handler

var DefaultMiddleware = []Middleware{
	WithRecover,
	WithMetrics,
	WithLogger,
	WithRequestID,
}

func WithMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	middlewares = append(DefaultMiddleware, middlewares...)

	withMiddleware := handler
	for i := range middlewares {
		withMiddleware = middlewares[i](withMiddleware)
	}
	return withMiddleware
}

// UserID extracts the authenticated API user's ID from the request context,
// placed there by WithAuthentication, reporting whether the request was
// authenticated.
func UserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}

// WithLocalhostOnly rejects requests that do not originate from a loopback address.
func WithLocalhostOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// WithAuthentication authenticates a request with a bearer token.
func WithAuthentication(database db.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || !strings.HasPrefix(token, snips.APIKeyTokenPrefix) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="snips.sh api"`)
			http.Error(w, "missing or malformed api key", http.StatusUnauthorized)
			return
		}

		key, err := database.FindAPIKeyByTokenHash(r.Context(), snips.HashAPIKeyToken(token))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if key == nil {
			w.Header().Set("WWW-Authenticate", `Bearer realm="snips.sh api"`)
			http.Error(w, "unknown api key", http.StatusUnauthorized)
			return
		}

		if key.IsExpired() {
			w.Header().Set("WWW-Authenticate", `Bearer realm="snips.sh api"`)
			http.Error(w, "expired api key", http.StatusUnauthorized)
			return
		}

		if err := database.TouchAPIKey(r.Context(), key.ID); err != nil {
			logger.From(r.Context()).Warn("unable to touch api key", "err", err, "api_key_id", key.ID)
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, key.UserID)
		next(w, r.WithContext(ctx))
	}
}

// WithRequestID adds a unique request ID to the request context, and echoes
// it as a response header so clients can reference it when reporting issues.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := id.New()
		r.Header.Set(RequestIDHeader, requestID)
		w.Header().Set(RequestIDHeader, requestID)

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

		reqLog := slog.With("svc", "http", "addr", addr, "request_id", requestID, "path", r.URL.Path)

		ctx := context.WithValue(r.Context(), logger.ContextKey, reqLog)
		reqLog.Info("connected")

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)

		reqLog.Info("disconnected", "dur", time.Since(start), "pattern", Pattern(r))
	})
}

// WithRecover will recover from any panics and log them.
func WithRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.From(r.Context()).Error("panic", "err", err)
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

		labels := []metrics.Label{
			{Name: "pattern", Value: Pattern(r)},
			{Name: "method", Value: r.Method},
		}

		metrics.IncrCounterWithLabels([]string{"http", "request"}, 1, labels)
		metrics.MeasureSinceWithLabels([]string{"http", "request", "duration"}, start, labels)
	})
}

func Pattern(r *http.Request) string {
	pattern := strings.TrimPrefix(r.Pattern, r.Method+" ")
	if pattern == "" {
		// empty pattern, didn't match router e.g. 404
		pattern = "*"
	}
	return pattern
}
