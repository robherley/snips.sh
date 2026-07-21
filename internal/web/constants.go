package web

type ContextKey string

const (
	RequestIDContextKey ContextKey = "request_id"
	UserIDContextKey    ContextKey = "user_id"

	RequestIDHeader = "X-Request-ID"
)
