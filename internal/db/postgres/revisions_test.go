package postgres

import (
	"strconv"
	"testing"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRevisions(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Create Revisions", "content")
		revisions := make([]*snips.Revision, 0, 3)

		for i := range 3 {
			revision := testutil.Fixtures.Revision(t)
			revision.FileID = file.ID
			revision.Size = uint64(i + 1)
			revision.Type = "markdown"
			require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("diff-"+strconv.Itoa(i+1)), 2))
			require.NotEmpty(t, revision.ID)
			require.Equal(t, int64(i+1), revision.Sequence)
			require.False(t, revision.CreatedAt.IsZero())
			revisions = append(revisions, &revision)
		}

		count, err := database.Revisions.CountByFileID(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		prunedDiff, err := database.Revisions.FindDiff(t.Context(), revisions[0].ID)
		require.NoError(t, err)
		require.Nil(t, prunedDiff)
	})

	t.Run("FindByFileID", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Find Revisions", "content")
		revisions := make([]*snips.Revision, 3)
		for i := range revisions {
			revision := testutil.Fixtures.Revision(t)
			revision.FileID = file.ID
			revision.Size = uint64(i + 1)
			revision.Type = "markdown"
			revisions[i] = &revision
			require.NoError(t, database.Revisions.Create(t.Context(), revisions[i], []byte("diff"), 0))
		}

		page, err := database.Revisions.FindByFileID(t.Context(), file.ID, db.WithLimit(2))
		require.NoError(t, err)
		require.Equal(t, []*snips.Revision{revisions[2], revisions[1]}, page)
		page, err = database.Revisions.FindByFileID(t.Context(), file.ID,
			db.WithLimit(2), db.WithCursor(db.Cursor{ID: revisions[1].ID}))
		require.NoError(t, err)
		require.Equal(t, []*snips.Revision{revisions[0]}, page)
		page, err = database.Revisions.FindByFileID(t.Context(), file.ID,
			db.WithCursor(db.Cursor{ID: "missing"}))
		require.NoError(t, err)
		require.Empty(t, page)
		page, err = database.Revisions.FindByFileID(t.Context(), "missing")
		require.NoError(t, err)
		require.Empty(t, page)
	})

	t.Run("FindByFileIDAndSequence", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Sequence", "content")
		revision := testutil.Fixtures.Revision(t)
		revision.FileID = file.ID
		require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("diff"), 0))

		foundRevision, err := database.Revisions.FindByFileIDAndSequence(t.Context(), file.ID, revision.Sequence)
		require.NoError(t, err)
		require.Equal(t, &revision, foundRevision)
		missingRevision, err := database.Revisions.FindByFileIDAndSequence(t.Context(), file.ID, 99)
		require.NoError(t, err)
		require.Nil(t, missingRevision)
	})

	t.Run("FindDiff", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Diff", "content")
		revision := testutil.Fixtures.Revision(t)
		revision.FileID = file.ID
		require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("a diff"), 0))

		diff, err := database.Revisions.FindDiff(t.Context(), revision.ID)
		require.NoError(t, err)
		require.Equal(t, []byte("a diff"), diff)
		diff, err = database.Revisions.FindDiff(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, diff)
	})

	t.Run("CountByFileID", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Count", "content")
		for range 2 {
			revision := testutil.Fixtures.Revision(t)
			revision.FileID = file.ID
			require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("diff"), 0))
		}

		count, err := database.Revisions.CountByFileID(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		count, err = database.Revisions.CountByFileID(t.Context(), "missing")
		require.NoError(t, err)
		require.Zero(t, count)
	})
}
