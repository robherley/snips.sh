package ssh

import (
	"time"
)

const (
	MaxSessionDuration = 15 * time.Minute

	MaxUploadSize    = 1 * 1024 * 1024 // 1MB
	UploadBufferSize = 1 * 1024        // 1KB

	LoggerContextKey      = "logger"
	RequestIDContextKey   = "request_id"
	FingerprintContextKey = "fingerprint"
	UserIDContextKey      = "user_id"

	FileRequestPrefix = "f:"
)
