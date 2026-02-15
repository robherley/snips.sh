package db

import (
	"context"

	"github.com/robherley/snips.sh/internal/snips"
)

//go:generate ../../script/mocks
type DB interface {
	// Migrate migrates the database.
	Migrate(ctx context.Context) error
	// FindFile returns a file by its ID. Includes file content.
	FindFile(ctx context.Context, id string) (*snips.File, error)
	// CreateFile creates a new file. If a user has more than maxFiles, an error is returned.
	CreateFile(ctx context.Context, file *snips.File, maxFiles uint64) error
	// UpdateFile updates a file.
	UpdateFile(ctx context.Context, file *snips.File) error
	// DeleteFile deletes a file by its ID.
	DeleteFile(ctx context.Context, id string) error
	// FindFilesByUser returns all files for a user. Does not include file content.
	FindFilesByUser(ctx context.Context, userID string) ([]*snips.File, error)
	// FindPublicKeyByFingerprint returns a public key by its fingerprint.
	FindPublicKeyByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error)
	// CreateUserWithPublicKey creates a new user with a public key.
	CreateUserWithPublicKey(ctx context.Context, publickey *snips.PublicKey) (*snips.User, error)
	// FindUser returns a user by its ID.
	FindUser(ctx context.Context, id string) (*snips.User, error)
	// CreateRevision creates a new file revision. If maxRevisions > 0, prunes oldest revisions exceeding the limit.
	CreateRevision(ctx context.Context, revision *snips.Revision, maxRevisions uint64) error
	// FindRevisionsByFileID returns all revisions for a file, ordered by id DESC. Does not include diff content.
	FindRevisionsByFileID(ctx context.Context, fileID string) ([]*snips.Revision, error)
	// FindRevision returns a revision by file ID and revision number, including diff content.
	FindRevision(ctx context.Context, fileID string, id int64) (*snips.Revision, error)
	// CountRevisionsByFileID returns the number of revisions for a file.
	CountRevisionsByFileID(ctx context.Context, fileID string) (int64, error)
	// DeleteRevisionsByFileID deletes all revisions for a file.
	DeleteRevisionsByFileID(ctx context.Context, fileID string) error
}
