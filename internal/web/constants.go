package web

type ContextKey string

const (
	RequestIDContextKey ContextKey = "request_id"

	RequestIDHeader = "X-Request-ID"
)
