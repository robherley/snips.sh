package web

type ctxKey int

const (
	requestIDContextKey ctxKey = iota
	fileContextKey
	signedContextKey
)

const RequestIDHeader = "X-Request-ID"
