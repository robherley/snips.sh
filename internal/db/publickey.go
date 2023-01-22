package db

import (
	"github.com/segmentio/ksuid"
)

type PublicKey struct {
	Model

	Fingerprint string `gorm:"index:pubkey_fingerprint,unique"`
	Type        string

	UserID ksuid.KSUID
	User   User
}
