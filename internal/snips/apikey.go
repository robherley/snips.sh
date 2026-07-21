package snips

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"encoding/hex"
	"time"
)

const (
	// APIKeyTokenPrefix prefixes every minted token so keys are recognizable
	// (and grep-able) in configs and secret scanners.
	APIKeyTokenPrefix = "snips_"

	apiKeyTokenBytes = 32
)

// APIKey grants REST API access as a user. Only a hash of the token is
// stored; the token itself is shown once at mint time.
type APIKey struct {
	ID         string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Name       string
	TokenHash  string
	UserID     string
	LastUsedAt *time.Time
	ExpiresAt  *time.Time // nil = never expires
}

// IsExpired reports whether the key's optional expiry has passed. Expired
// keys are rejected for authentication but kept (and listed) until removed.
func (k *APIKey) IsExpired() bool {
	return k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt)
}

// DisplayName returns the key's name, falling back to its ID when unnamed.
func (k *APIKey) DisplayName() string {
	if k.Name != "" {
		return k.Name
	}
	return k.ID
}

// NewAPIKeyToken mints a new random API token and returns it alongside the
// hash to persist.
func NewAPIKeyToken() (token string, hash string, err error) {
	raw := make([]byte, apiKeyTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	token = APIKeyTokenPrefix + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return token, HashAPIKeyToken(token), nil
}

// HashAPIKeyToken returns the hex-encoded SHA-384 digest of a token, the only
// form in which tokens are stored.
//
// A fast hash is deliberate: unlike a password, a 256-bit random token can't
// be brute forced, and lookups stay indexable.
func HashAPIKeyToken(token string) string {
	digest := sha512.Sum384([]byte(token))
	return hex.EncodeToString(digest[:])
}
