package ssh

import (
	"github.com/charmbracelet/ssh"
	"github.com/segmentio/ksuid"
)

type UserSession struct {
	ssh.Session
}

func (sesh *UserSession) UserID() ksuid.KSUID {
	return sesh.Context().Value(UserIDContextKey).(ksuid.KSUID)
}

func (sesh *UserSession) PublicKeyFingerprint() string {
	return sesh.Context().Value(FingerprintContextKey).(string)
}
