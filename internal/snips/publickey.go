package snips

import "time"

type PublicKey struct {
	ID          string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Fingerprint string
	Type        string
	UserID      string
}
