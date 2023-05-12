package ssh

const (
	UploadBufferSize = 1 * 1024 // 1KB

	LoggerContextKey      = "logger"
	RequestIDContextKey   = "request_id"
	FingerprintContextKey = "fingerprint"
	UserIDContextKey      = "user_id"

	FileRequestPrefix = "f:"
)
