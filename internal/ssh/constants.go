package ssh

import (
	"time"
)

const (
	MaxTimeout  = 5 * time.Minute
	IdleTimeout = 30 * time.Second

	MaxUploadSize    = 1 * 1024 * 1024 // 1MB
	UploadBufferSize = 1 * 1024        // 1KB

	LoggerContextKey      = "logger"
	RequestIDContextKey   = "request_id"
	FingerprintContextKey = "fingerprint"
	UserIDContextKey      = "user_id"

	FileRequestPrefix = "f:"
)
