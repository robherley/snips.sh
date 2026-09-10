package testutil

import (
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type fixtures struct{}

var Fixtures = &fixtures{}

func (f *fixtures) File(t *testing.T) snips.File {
	t.Helper()

	file := snips.File{
		ID:        id.New(),
		CreatedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now().Add(-5 * time.Minute),
		Size:      100,
		Private:   false,
		UserID:    id.New(),
	}

	return file
}

func (f *fixtures) PublicKey(t *testing.T) snips.PublicKey {
	t.Helper()

	now := time.Now().UTC().Add(-5 * time.Minute)
	return snips.PublicKey{
		ID:          id.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Fingerprint: "SHA256:" + id.New(),
		Type:        "ssh-ed25519",
		UserID:      id.New(),
	}
}

func (f *fixtures) Revision(t *testing.T) snips.Revision {
	t.Helper()

	return snips.Revision{
		ID:        id.New(),
		Sequence:  1,
		FileID:    id.New(),
		CreatedAt: time.Now().UTC().Add(-5 * time.Minute),
		Size:      100,
		Type:      "plaintext",
	}
}

func (f *fixtures) APIKey(t *testing.T) snips.APIKey {
	t.Helper()

	now := time.Now().UTC().Add(-5 * time.Minute)
	return snips.APIKey{
		ID:        id.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      "test",
		TokenHash: id.New(),
		UserID:    id.New(),
	}
}
