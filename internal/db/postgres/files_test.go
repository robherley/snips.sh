package postgres_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestFiles(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := testutil.Fixtures.File(t)
		file.UserID = user.ID
		file.Type = "plaintext"
		file.Name = "First"
		file.Private = true

		require.NoError(t, database.Files.Create(t.Context(), &file, []byte("first content"), 1))
		require.NotEmpty(t, file.ID)
		require.False(t, file.CreatedAt.IsZero())
		require.Equal(t, file.CreatedAt, file.UpdatedAt)
		require.Equal(t, uint64(len("first content")), file.Size)

		var internalID int64
		var displayID string
		err := database.SQL.QueryRowContext(t.Context(),
			`SELECT id, display_id FROM `+database.Schema+`.files WHERE display_id = $1`, file.ID,
		).Scan(&internalID, &displayID)
		require.NoError(t, err)
		require.Positive(t, internalID)
		require.Equal(t, file.ID, displayID)

		duplicate := testutil.Fixtures.File(t)
		duplicate.UserID = user.ID
		duplicate.Name = "fIRST"
		err = database.Files.Create(t.Context(), &duplicate, nil, 0)
		require.ErrorIs(t, err, db.ErrNameTaken)
		overLimit := testutil.Fixtures.File(t)
		overLimit.UserID = user.ID
		err = database.Files.Create(t.Context(), &overLimit, nil, 1)
		require.ErrorIs(t, err, db.ErrFileLimit)
	})

	t.Run("Find", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Find", "content")

		foundFile, err := database.Files.Find(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, file, foundFile)
		missingFile, err := database.Files.Find(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, missingFile)
	})

	t.Run("FindWithContent", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "With Content", "content")

		foundFile, content, err := database.Files.FindWithContent(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, file, foundFile)
		require.Equal(t, []byte("content"), content)
		foundFile, content, err = database.Files.FindWithContent(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, foundFile)
		require.Nil(t, content)
	})

	t.Run("FindContent", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Content", "content")

		content, err := database.Files.FindContent(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, []byte("content"), content)
		content, err = database.Files.FindContent(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, content)
	})

	t.Run("Update", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "First", "original content")
		database.createTestFile(t, user.ID, "Taken", "content")
		file.Private = true
		file.Type = "markdown"
		file.Name = "Renamed"

		require.NoError(t, database.Files.Update(t.Context(), file))
		foundFile, content, err := database.Files.FindWithContent(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, file, foundFile)
		require.Equal(t, []byte("original content"), content, "metadata updates must preserve content")

		file.Name = "tAKEN"
		err = database.Files.Update(t.Context(), file)
		require.ErrorIs(t, err, db.ErrNameTaken)
	})

	t.Run("UpdateContent", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "First", "original content")
		database.createTestFile(t, user.ID, "Taken", "content")
		file.Private = true
		file.Type = "markdown"
		file.Name = "Updated"

		require.NoError(t, database.Files.UpdateContent(t.Context(), file, []byte("new content")))
		require.Equal(t, uint64(len("new content")), file.Size)
		foundFile, content, err := database.Files.FindWithContent(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, file, foundFile)
		require.Equal(t, []byte("new content"), content)

		file.Name = "tAKEN"
		err = database.Files.UpdateContent(t.Context(), file, []byte("rejected content"))
		require.ErrorIs(t, err, db.ErrNameTaken)
		content, err = database.Files.FindContent(t.Context(), file.ID)
		require.NoError(t, err)
		require.Equal(t, []byte("new content"), content)
	})

	t.Run("FindByUser", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		files := make([]*snips.File, 3)
		for i, name := range []string{"First", "Second", "Third"} {
			file := testutil.Fixtures.File(t)
			file.UserID = user.ID
			file.Type = "plaintext"
			file.Name = name
			files[i] = &file
			require.NoError(t, database.Files.Create(t.Context(), files[i], []byte(name), 0))
		}
		otherPublicKey := testutil.Fixtures.PublicKey(t)
		otherUser, err := database.Users.CreateWithPublicKey(t.Context(), &otherPublicKey)
		require.NoError(t, err)
		database.createTestFile(t, otherUser.ID, "Other", "content")

		filePage, err := database.Files.FindByUser(t.Context(), user.ID, db.WithLimit(2))
		require.NoError(t, err)
		require.Equal(t, []*snips.File{files[2], files[1]}, filePage)
		filePage, err = database.Files.FindByUser(t.Context(), user.ID,
			db.WithLimit(2), db.WithCursor(db.Cursor{ID: files[1].ID}))
		require.NoError(t, err)
		require.Equal(t, []*snips.File{files[0]}, filePage)
		filePage, err = database.Files.FindByUser(t.Context(), user.ID,
			db.WithCursor(db.Cursor{ID: "missing"}))
		require.NoError(t, err)
		require.Empty(t, filePage)
	})

	t.Run("FindByName", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Named File", "content")

		foundFile, err := database.Files.FindByName(t.Context(), user.ID, "nAMED fILE")
		require.NoError(t, err)
		require.Equal(t, file, foundFile)
		missingFile, err := database.Files.FindByName(t.Context(), user.ID, "missing")
		require.NoError(t, err)
		require.Nil(t, missingFile)
		missingFile, err = database.Files.FindByName(t.Context(), "other-user", file.Name)
		require.NoError(t, err)
		require.Nil(t, missingFile)
	})

	t.Run("CountByUser", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		database.createTestFile(t, user.ID, "First", "content")
		database.createTestFile(t, user.ID, "Second", "content")

		count, err := database.Files.CountByUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		count, err = database.Files.CountByUser(t.Context(), "missing")
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("Delete", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		file := database.createTestFile(t, user.ID, "Delete", "content")
		revision := testutil.Fixtures.Revision(t)
		revision.FileID = file.ID
		require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("diff"), 0))

		require.NoError(t, database.Files.Delete(t.Context(), file.ID))
		missingFile, err := database.Files.Find(t.Context(), file.ID)
		require.NoError(t, err)
		require.Nil(t, missingFile)
		count, err := database.Revisions.CountByFileID(t.Context(), file.ID)
		require.NoError(t, err)
		require.Zero(t, count)
		require.NoError(t, database.Files.Delete(t.Context(), "missing"))
	})

	t.Run("DeleteByUser", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		firstFile := database.createTestFile(t, user.ID, "First", "content")
		secondFile := database.createTestFile(t, user.ID, "Second", "content")
		for _, file := range []*snips.File{firstFile, secondFile} {
			revision := testutil.Fixtures.Revision(t)
			revision.FileID = file.ID
			require.NoError(t, database.Revisions.Create(t.Context(), &revision, []byte("diff"), 0))
		}
		otherPublicKey := testutil.Fixtures.PublicKey(t)
		otherUser, err := database.Users.CreateWithPublicKey(t.Context(), &otherPublicKey)
		require.NoError(t, err)
		otherFile := database.createTestFile(t, otherUser.ID, "Other", "content")

		deletedFiles, err := database.Files.DeleteByUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), deletedFiles)
		count, err := database.Files.CountByUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Zero(t, count)
		for _, file := range []*snips.File{firstFile, secondFile} {
			count, err = database.Revisions.CountByFileID(t.Context(), file.ID)
			require.NoError(t, err)
			require.Zero(t, count)
		}
		foundOtherFile, err := database.Files.Find(t.Context(), otherFile.ID)
		require.NoError(t, err)
		require.Equal(t, otherFile, foundOtherFile)

		deletedFiles, err = database.Files.DeleteByUser(t.Context(), "missing")
		require.NoError(t, err)
		require.Zero(t, deletedFiles)
	})
}
