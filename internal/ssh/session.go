package ssh

import (
	"strings"

	"github.com/charmbracelet/ssh"
)

type UserSession struct {
	ssh.Session
}

func (sesh *UserSession) UserID() string {
	return sesh.Context().Value(UserIDContextKey).(string)
}

func (sesh *UserSession) PublicKeyFingerprint() string {
	return sesh.Context().Value(FingerprintContextKey).(string)
}

func (sesh *UserSession) RequestID() string {
	return sesh.Context().Value(RequestIDContextKey).(string)
}

func (sesh *UserSession) IsFileRequest() bool {
	return strings.HasPrefix(sesh.User(), FileRequestPrefix)
}

func (sesh *UserSession) RequestedFileID() string {
	return strings.TrimPrefix(sesh.User(), FileRequestPrefix)
}

func (sesh *UserSession) IsPTY() bool {
	_, _, isPty := sesh.Pty()
	return isPty
}
