package db

import "github.com/robherley/snips.sh/internal/db/models"

type DB interface {
	// Migrate runs the database migrations.
	Migrate() error
	// File returns a file by its ID.
	File(id string) (*models.File, error)
	// NewFile creates a new file.
	NewFile(file *models.File) error
	// UpdateFile updates a file.
	UpdateFile(file *models.File) error
	// DeleteFile deletes a file by its ID.
	DeleteFile(id string) error
	// FileForUser returns a file by its ID and user ID.
	FileForUser(id, userID string) (*models.File, error)
	// FilesForUser returns all files for a user.
	// If withContent is false, the content field will be omitted.
	FilesForUser(userID string, withContent bool) ([]models.File, error)
	// PublicKeyForFingerprint returns a public key by its fingerprint.
	PublicKeyForFingerprint(fingerprint string) (*models.PublicKey, error)
	// NewUser creates a new user with a public key.
	NewUser(publickey *models.PublicKey) (*models.User, error)
	// User returns a user by its ID.
	User(id string) (*models.User, error)
}
