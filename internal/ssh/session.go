package ssh

import (
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
