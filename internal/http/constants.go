package http

type ContextKey string

const (
	LoggerContextKey    ContextKey = "logger"
	RequestIDContextKey ContextKey = "request_id"

	RequestIDHeader = "X-Request-ID"
)
