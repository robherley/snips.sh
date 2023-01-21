package ssh

import (
	"time"

	"github.com/robherley/snips.sh/internal/file"
)

const (
	MaxTimeout  = 5 * time.Minute
	IdleTimeout = 30 * time.Second

	MaxUploadSize    = int64(512 * file.KB)
	UploadBufferSize = int64(1 * file.KB)

	LoggerContextKey      = "logger"
	RequestIDContextKey   = "request_id"
	FingerprintContextKey = "fingerprint"
	UserIDContextKey      = "user_id"
)
