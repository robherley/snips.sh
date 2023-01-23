package db

type PublicKey struct {
	Model

	Fingerprint string `gorm:"index:pubkey_fingerprint,unique"`
	Type        string

	UserID string
	User   User
}
