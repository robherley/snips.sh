package web

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/signer"
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

// WithRequestID adds a unique request ID to the request context.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := id.New()
		r.Header.Set(RequestIDHeader, requestID)

		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
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

// WithFile is a per-route middleware that extracts a file from the {fileID} path
// parameter, verifies access for private files via URL signature, and stores
// the file in the request context for downstream handlers.
func WithFile(cfg *config.Config, database db.DB, next http.HandlerFunc) http.HandlerFunc {
	sgnr := signer.New(cfg.HMACKey)
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		fileID := r.PathValue("fileID")
		if fileID == "" {
			http.NotFound(w, r)
			return
		}

		file, err := database.FindFile(r.Context(), fileID)
		if err != nil {
			log.Error("unable to lookup file", "err", err)
			http.NotFound(w, r)
			return
		}

		if file == nil {
			http.NotFound(w, r)
			return
		}

		isSigned := sgnr.VerifyURLAndNotExpired(*r.URL)

		if file.Private && !isSigned {
			http.NotFound(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), fileContextKey, file)
		ctx = context.WithValue(ctx, signedContextKey, isSigned)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// FileFrom retrieves the *snips.File stored in the context by WithFile.
func FileFrom(ctx context.Context) *snips.File {
	file, _ := ctx.Value(fileContextKey).(*snips.File)
	return file
}

// IsSignedRequest returns whether the request had a valid, non-expired URL signature.
func IsSignedRequest(ctx context.Context) bool {
	signed, _ := ctx.Value(signedContextKey).(bool)
	return signed
}
