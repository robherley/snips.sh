package models

type PublicKey struct {
	Base

	Fingerprint string `gorm:"index:pubkey_fingerprint,unique"`
	Type        string

	UserID string
	User   User
}
