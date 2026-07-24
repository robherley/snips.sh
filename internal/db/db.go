package db

import (
	"context"
	"io"

	"github.com/robherley/snips.sh/internal/snips"
)

// DB groups database operations by their owning table.
type DB struct {
	Migrator
	io.Closer
	Files      Files
	PublicKeys PublicKeys
	Users      Users
	Revisions  Revisions
	APIKeys    APIKeys
}

type Migrator interface {
	// Migrate migrates the database.
	Migrate(ctx context.Context) error
}

type Files interface {
	// Find returns a file by its ID. It does not include file content.
	Find(ctx context.Context, id string) (*snips.File, error)
	// Create creates a new file. If a user has more than maxFiles, an error is returned.
	Create(ctx context.Context, file *snips.File, content []byte, compress bool, maxFiles uint64) error
	// GetContent returns a file's decompressed content by ID.
	GetContent(ctx context.Context, id string) ([]byte, error)
	// Update updates a file.
	Update(ctx context.Context, file *snips.File) error
	// UpdateContent updates a file and replaces its content.
	UpdateContent(ctx context.Context, file *snips.File, content []byte, compress bool) error
	// Delete deletes a file by its ID.
	Delete(ctx context.Context, id string) error
	// DeleteByUser deletes all of a user's files and their revisions, returning the number of files deleted.
	DeleteByUser(ctx context.Context, userID string) (int64, error)
	// FindByUser returns a user's files, newest first. It does not include file content.
	FindByUser(ctx context.Context, userID string, opts ...PageOption) ([]*snips.File, error)
	// FindByName returns a user's file with the given name (case-insensitive). It does not include file content.
	FindByName(ctx context.Context, userID, name string) (*snips.File, error)
	// CountByUser returns the number of files a user has.
	CountByUser(ctx context.Context, userID string) (int64, error)
}

type PublicKeys interface {
	// FindByFingerprint returns a public key by its fingerprint.
	FindByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error)
}

type Users interface {
	// CreateWithPublicKey creates a new user with a public key.
	CreateWithPublicKey(ctx context.Context, publickey *snips.PublicKey) (*snips.User, error)
	// Find returns a user by its ID.
	Find(ctx context.Context, id string) (*snips.User, error)
	// Update updates a user's mutable fields (currently theme color and updated_at).
	Update(ctx context.Context, user *snips.User) error
}

type Revisions interface {
	// Create creates a new file revision. If maxRevisions > 0, prunes oldest revisions exceeding the limit.
	Create(ctx context.Context, revision *snips.Revision, diff []byte, compress bool, maxRevisions uint64) error
	// GetDiff returns a revision's decompressed diff by ID.
	GetDiff(ctx context.Context, id string) ([]byte, error)
	// FindByFileID returns a file's revisions, newest first. It does not include diff content.
	FindByFileID(ctx context.Context, fileID string, opts ...PageOption) ([]*snips.Revision, error)
	// FindByFileIDAndSequence returns a revision by file ID and sequence number. It does not include diff content.
	FindByFileIDAndSequence(ctx context.Context, fileID string, sequence int64) (*snips.Revision, error)
	// CountByFileID returns the number of revisions for a file.
	CountByFileID(ctx context.Context, fileID string) (int64, error)
}

type APIKeys interface {
	// Create creates a new API key. If a user has maxKeys or more keys, ErrAPIKeyLimit is returned.
	Create(ctx context.Context, key *snips.APIKey, maxKeys uint64) error
	// FindByTokenHash returns an API key by its token hash.
	FindByTokenHash(ctx context.Context, tokenHash string) (*snips.APIKey, error)
	// FindByUser returns all API keys for a user, newest first.
	FindByUser(ctx context.Context, userID string) ([]*snips.APIKey, error)
	// Delete deletes a user's API key by ID, reporting whether a key was deleted.
	Delete(ctx context.Context, id, userID string) (bool, error)
	// Touch updates an API key's last_used_at timestamp.
	Touch(ctx context.Context, id string) error
}
