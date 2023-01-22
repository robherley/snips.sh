package db

type User struct {
	Model

	// Alias is an alternative (optional) name for the user
	Alias *string `gorm:"index:user_alias,unique"`

	// PublicKeys are all the user's public keys
	PublicKeys []PublicKey
}
