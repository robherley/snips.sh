package dbmock

import (
	"github.com/robherley/snips.sh/internal/db"
	"github.com/stretchr/testify/mock"
)

// Database bundles a DB with its table-level mocks.
type Database struct {
	DB         *db.DB
	Migrator   *MockMigrator
	Files      *MockFiles
	PublicKeys *MockPublicKeys
	Users      *MockUsers
	Revisions  *MockRevisions
	APIKeys    *MockAPIKeys
}

// NewDB creates a database composed of independently mockable table stores.
func NewDB(t interface {
	mock.TestingT
	Cleanup(func())
}) *Database {
	mocks := &Database{
		Migrator:   NewMockMigrator(t),
		Files:      NewMockFiles(t),
		PublicKeys: NewMockPublicKeys(t),
		Users:      NewMockUsers(t),
		Revisions:  NewMockRevisions(t),
		APIKeys:    NewMockAPIKeys(t),
	}
	mocks.DB = &db.DB{
		Migrator:   mocks.Migrator,
		Files:      mocks.Files,
		PublicKeys: mocks.PublicKeys,
		Users:      mocks.Users,
		Revisions:  mocks.Revisions,
		APIKeys:    mocks.APIKeys,
	}

	return mocks
}
