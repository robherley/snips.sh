package db

import (
	"github.com/segmentio/ksuid"
)

type PublicKey struct {
	Model

	// Fingerprint is the public key's fingerprint
	Fingerprint string `gorm:"index:pubkey_fingerprint,unique"`
	// Type is the public key's type (e.g. ssh-rsa)
	Type string

	// UserID is the ID of the user this public key belongs to
	UserID ksuid.KSUID
	User   User
}
