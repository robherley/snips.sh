package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/stretchr/testify/suite"
)

type SqliteSuite struct {
	suite.Suite

	testDB *sql.DB
}

func TestSqliteSuite(t *testing.T) {
	suite.Run(t, new(SqliteSuite))
}

func (s *SqliteSuite) getTestDB(migrate bool) *db.Sqlite {
	database := &db.Sqlite{DB: s.testDB}

	if migrate {
		err := database.Migrate(context.TODO())
		s.Require().NoError(err)
	}

	return database
}

func (s *SqliteSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)
	s.testDB = db
}

func (s *SqliteSuite) TestMigrate() {
	database := s.getTestDB(false)

	err := database.Migrate(context.TODO())
	s.Require().NoError(err)

	rows, err := s.testDB.Query("SELECT name FROM sqlite_master WHERE type='table'")
	s.Require().NoError(err)

	var tables []string
	for rows.Next() {
		var name string
		s.Require().NoError(rows.Scan(&name))
		tables = append(tables, name)
	}

	s.Require().Contains(tables, "users")
	s.Require().Contains(tables, "files")
	s.Require().Contains(tables, "public_keys")
}

func (s *SqliteSuite) TestFindFile() {
	database := s.getTestDB(true)

	existingFile := &snips.File{
		ID:         id.New(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		Size:       11,
		RawContent: []byte("hello world"),
		Private:    false,
		Type:       "plaintext",
		UserID:     id.New(),
	}

	const query = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.testDB.Exec(
		query,
		existingFile.ID,
		existingFile.CreatedAt,
		existingFile.UpdatedAt,
		existingFile.Size,
		existingFile.RawContent,
		existingFile.Private,
		existingFile.Type,
		existingFile.UserID,
	)
	s.Require().NoError(err)

	file, err := database.FindFile(context.TODO(), existingFile.ID)
	s.Require().NoError(err)
	s.Require().Equal(existingFile, file)
}

func (s *SqliteSuite) TestFindFile_DoesNotExist() {
	database := s.getTestDB(true)

	file, err := database.FindFile(context.TODO(), id.New())
	s.Require().NoError(err)
	s.Require().Nil(file)
}

func (s *SqliteSuite) TestCreateFile() {
	database := s.getTestDB(true)

	file := &snips.File{
		Size:       11,
		RawContent: []byte("hello world"),
		Private:    false,
		Type:       "plaintext",
		UserID:     id.New(),
	}

	err := database.CreateFile(context.TODO(), file, 1337)
	s.Require().NoError(err)

	s.Require().NotEmpty(file.ID)
	s.Require().NotEmpty(file.CreatedAt)
	s.Require().NotEmpty(file.UpdatedAt)

	var (
		id        string
		createdAt time.Time
		updatedAt time.Time
		size      uint64
		content   []byte
		private   bool
		fileType  string
		userID    string
	)

	row := s.testDB.QueryRow("SELECT id, created_at, updated_at, size, content, private, type, user_id FROM files")
	err = row.Scan(&id, &createdAt, &updatedAt, &size, &content, &private, &fileType, &userID)
	s.Require().NoError(err)

	s.Require().Equal(file.ID, id)
	s.Require().Equal(file.CreatedAt, createdAt)
	s.Require().Equal(file.UpdatedAt, updatedAt)
	s.Require().Equal(file.Size, size)
	s.Require().Equal(file.RawContent, content)
	s.Require().Equal(file.Private, private)
	s.Require().Equal(file.Type, fileType)
	s.Require().Equal(file.UserID, userID)
}

func (s *SqliteSuite) TestCreateFile_FileLimit() {
	database := s.getTestDB(true)

	userID := id.New()

	maxFiles := uint64(5)

	for i := uint64(0); i < maxFiles; i++ {
		err := database.CreateFile(context.TODO(), &snips.File{
			Size:       11,
			RawContent: []byte("hello world"),
			Private:    false,
			Type:       "plaintext",
			UserID:     userID,
		}, maxFiles)
		s.Require().NoError(err)
	}

	err := database.CreateFile(context.TODO(), &snips.File{
		Size:       11,
		RawContent: []byte("should fail"),
		Private:    false,
		Type:       "plaintext",
		UserID:     userID,
	}, maxFiles)

	s.Require().ErrorIs(err, db.ErrFileLimit)
}

func (s *SqliteSuite) TestUpdateFile() {
	database := s.getTestDB(true)

	originalUpdatedAt := time.Now().UTC()

	existingFile := &snips.File{
		ID:         id.New(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  originalUpdatedAt,
		Size:       11,
		RawContent: []byte("hello world"),
		Private:    false,
		Type:       "plaintext",
		UserID:     id.New(),
	}

	const query = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.testDB.Exec(
		query,
		existingFile.ID,
		existingFile.CreatedAt,
		existingFile.UpdatedAt,
		existingFile.Size,
		existingFile.RawContent,
		existingFile.Private,
		existingFile.Type,
		existingFile.UserID,
	)
	s.Require().NoError(err)

	existingFile.Size = 22
	existingFile.RawContent = []byte("hello world hello world")
	existingFile.Private = true
	existingFile.Type = "markdown"

	err = database.UpdateFile(context.TODO(), existingFile)
	s.Require().NoError(err)

	var (
		id        string
		createdAt time.Time
		updatedAt time.Time
		size      uint64
		content   []byte
		private   bool
		fileType  string
		userID    string
	)

	row := s.testDB.QueryRow("SELECT id, created_at, updated_at, size, content, private, type, user_id FROM files")
	err = row.Scan(&id, &createdAt, &updatedAt, &size, &content, &private, &fileType, &userID)
	s.Require().NoError(err)

	s.Require().Equal(existingFile.ID, id)
	s.Require().Equal(existingFile.CreatedAt, createdAt)
	s.Require().NotEqual(originalUpdatedAt, updatedAt)
	s.Require().Equal(existingFile.Size, size)
	s.Require().Equal(existingFile.RawContent, content)
	s.Require().Equal(existingFile.Private, private)
	s.Require().Equal(existingFile.Type, fileType)
	s.Require().Equal(existingFile.UserID, userID)
}

func (s *SqliteSuite) TestDeleteFile() {
	database := s.getTestDB(true)

	existingFile := &snips.File{
		ID:         id.New(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		Size:       11,
		RawContent: []byte("hello world"),
		Private:    false,
		Type:       "plaintext",
		UserID:     id.New(),
	}

	const query = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.testDB.Exec(
		query,
		existingFile.ID,
		existingFile.CreatedAt,
		existingFile.UpdatedAt,
		existingFile.Size,
		existingFile.RawContent,
		existingFile.Private,
		existingFile.Type,
		existingFile.UserID,
	)
	s.Require().NoError(err)

	err = database.DeleteFile(context.TODO(), existingFile.ID)
	s.Require().NoError(err)

	row := s.testDB.QueryRow("SELECT id FROM files")
	err = row.Scan(&existingFile.ID)
	s.Require().Equal(sql.ErrNoRows, err)
}

func (s *SqliteSuite) TestDeleteFilesByUser() {
	database := s.getTestDB(true)

	userID := id.New()
	otherUserID := id.New()

	const insertFileQuery = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	const insertRevisionQuery = `
		INSERT INTO revisions (
			id,
			sequence,
			file_id,
			created_at,
			diff,
			size,
			type
		) VALUES (?, ?, ?, ?, ?, ?, ?)`

	insertFile := func(ownerID string) string {
		fileID := id.New()
		_, err := s.testDB.Exec(
			insertFileQuery,
			fileID,
			time.Now().UTC(),
			time.Now().UTC(),
			11,
			[]byte("hello world"),
			false,
			"plaintext",
			ownerID,
		)
		s.Require().NoError(err)

		_, err = s.testDB.Exec(
			insertRevisionQuery,
			id.New(),
			1,
			fileID,
			time.Now().UTC(),
			[]byte("diff"),
			4,
			"create",
		)
		s.Require().NoError(err)

		return fileID
	}

	numFiles := 3
	for range numFiles {
		insertFile(userID)
	}
	otherFileID := insertFile(otherUserID)

	count, err := database.DeleteFilesByUser(context.TODO(), userID)
	s.Require().NoError(err)
	s.Require().Equal(int64(numFiles), count)

	var remainingFiles int
	err = s.testDB.QueryRow("SELECT COUNT(*) FROM files").Scan(&remainingFiles)
	s.Require().NoError(err)
	s.Require().Equal(1, remainingFiles)

	var remainingRevisions int
	err = s.testDB.QueryRow("SELECT COUNT(*) FROM revisions WHERE file_id != ?", otherFileID).Scan(&remainingRevisions)
	s.Require().NoError(err)
	s.Require().Equal(0, remainingRevisions)
}

func (s *SqliteSuite) TestFindFilesByUser() {
	database := s.getTestDB(true)

	userID := id.New()

	numFiles := 10

	for i := 0; i < numFiles; i++ {
		existingFile := &snips.File{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world"),
			Private:    false,
			Type:       "plaintext",
			UserID:     userID,
		}

		const query = `
			INSERT INTO files (
				id,
				created_at,
				updated_at,
				size,
				content,
				private,
				type,
				user_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err := s.testDB.Exec(
			query,
			existingFile.ID,
			existingFile.CreatedAt,
			existingFile.UpdatedAt,
			existingFile.Size,
			existingFile.RawContent,
			existingFile.Private,
			existingFile.Type,
			existingFile.UserID,
		)
		s.Require().NoError(err)
	}

	files, err := database.FindFilesByUser(context.TODO(), userID)
	s.Require().NoError(err)

	s.Require().Equal(numFiles, len(files))

	for _, file := range files {
		s.Require().Equal(userID, file.UserID)
		s.Require().Empty(file.RawContent)
	}
}

func (s *SqliteSuite) TestCountFilesByUser() {
	database := s.getTestDB(true)

	userID := id.New()

	count, err := database.CountFilesByUser(context.TODO(), userID)
	s.Require().NoError(err)
	s.Require().Equal(int64(0), count)

	numFiles := 3
	for range numFiles {
		const query = `
			INSERT INTO files (
				id,
				created_at,
				updated_at,
				size,
				content,
				private,
				type,
				user_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err := s.testDB.Exec(
			query,
			id.New(),
			time.Now().UTC(),
			time.Now().UTC(),
			11,
			[]byte("hello world"),
			false,
			"plaintext",
			userID,
		)
		s.Require().NoError(err)
	}

	count, err = database.CountFilesByUser(context.TODO(), userID)
	s.Require().NoError(err)
	s.Require().Equal(int64(numFiles), count)

	count, err = database.CountFilesByUser(context.TODO(), id.New())
	s.Require().NoError(err)
	s.Require().Equal(int64(0), count)
}

func (s *SqliteSuite) TestFindFilesByUser_DoesNotExist() {
	database := s.getTestDB(true)

	files, err := database.FindFilesByUser(context.TODO(), id.New())
	s.Require().NoError(err)
	s.Require().NotNil(files)
	s.Require().Empty(files)
}

func (s *SqliteSuite) TestFindPublicKeyByFingerprint() {
	database := s.getTestDB(true)

	userID := id.New()
	fingerprint := "SHA256:J+Xp9z5tPlVHlZJLFGmNUknvCWoAOGsOjioDs5UHDa4"

	existingPublicKey := &snips.PublicKey{
		ID:          id.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Fingerprint: fingerprint,
		Type:        "ssh-ed25519",
		UserID:      userID,
	}

	const query = `
		INSERT INTO public_keys (
			id,
			created_at,
			updated_at,
			fingerprint,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?)`

	_, err := database.Exec(
		query,
		existingPublicKey.ID,
		existingPublicKey.CreatedAt,
		existingPublicKey.UpdatedAt,
		existingPublicKey.Fingerprint,
		existingPublicKey.Type,
		existingPublicKey.UserID,
	)
	s.Require().NoError(err)

	publicKey, err := database.FindPublicKeyByFingerprint(context.Background(), fingerprint)
	s.Require().NoError(err)

	s.Require().Equal(existingPublicKey, publicKey)
}

func (s *SqliteSuite) TestFindPublicKeyByFingerprint_DoesNotExist() {
	database := s.getTestDB(true)

	pk, err := database.FindPublicKeyByFingerprint(context.TODO(), id.New())
	s.Require().NoError(err)
	s.Require().Nil(pk)
}

func (s *SqliteSuite) TestCreateUserWithPublicKey() {
	database := s.getTestDB(true)

	newPublicKey := &snips.PublicKey{
		Fingerprint: "SHA256:J+Xp9z5tPlVHlZJLFGmNUknvCWoAOGsOjioDs5UHDa4",
		Type:        "ssh-ed25519",
	}

	user, err := database.CreateUserWithPublicKey(context.Background(), newPublicKey)
	s.Require().NoError(err)

	{
		var (
			id        string
			createdAt time.Time
			updatedAt time.Time
		)
		row := s.testDB.QueryRow("SELECT id, created_at, updated_at FROM users")
		err = row.Scan(&id, &createdAt, &updatedAt)
		s.Require().NoError(err)

		s.Require().Equal(user.ID, id)
		s.Require().Equal(user.CreatedAt, createdAt)
		s.Require().Equal(user.UpdatedAt, updatedAt)
	}

	{
		var (
			id          string
			createdAt   time.Time
			updatedAt   time.Time
			fingerprint string
			keyType     string
			userID      string
		)

		row := s.testDB.QueryRow("SELECT id, created_at, updated_at, fingerprint, type, user_id FROM public_keys")
		err = row.Scan(&id, &createdAt, &updatedAt, &fingerprint, &keyType, &userID)
		s.Require().NoError(err)

		s.Require().Equal(newPublicKey.ID, id)
		s.Require().Equal(newPublicKey.CreatedAt, createdAt)
		s.Require().Equal(newPublicKey.UpdatedAt, updatedAt)
		s.Require().Equal(newPublicKey.Fingerprint, fingerprint)
		s.Require().Equal(newPublicKey.Type, keyType)
		s.Require().Equal(newPublicKey.UserID, userID)
	}

	s.Require().NotEmpty(user.ID)
	s.Require().NotEmpty(user.CreatedAt)
	s.Require().NotEmpty(user.UpdatedAt)
	s.Require().Equal(newPublicKey.UserID, user.ID)
}

func (s *SqliteSuite) TestFindUser() {
	database := s.getTestDB(true)

	userID := id.New()

	existingUser := &snips.User{
		ID:        userID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	const query = `
		INSERT INTO users (
			id,
			created_at,
			updated_at
		) VALUES (?, ?, ?)`

	_, err := database.Exec(
		query,
		existingUser.ID,
		existingUser.CreatedAt,
		existingUser.UpdatedAt,
	)
	s.Require().NoError(err)

	user, err := database.FindUser(context.Background(), userID)
	s.Require().NoError(err)

	s.Require().Equal(existingUser, user)
}

func (s *SqliteSuite) TestFindUser_DoesNotExist() {
	database := s.getTestDB(true)

	user, err := database.FindUser(context.Background(), id.New())
	s.Require().NoError(err)
	s.Require().Nil(user)
}

func (s *SqliteSuite) createFile(database *db.Sqlite, name string) *snips.File {
	file := &snips.File{
		Size:       11,
		RawContent: []byte("hello world"),
		Private:    false,
		Type:       "plaintext",
		UserID:     id.New(),
		Name:       name,
	}

	err := database.CreateFile(context.Background(), file, 0)
	s.Require().NoError(err)

	return file
}

func (s *SqliteSuite) TestCreateFile_WithName() {
	database := s.getTestDB(true)

	file := s.createFile(database, "deploy-notes")

	found, err := database.FindFile(context.Background(), file.ID)
	s.Require().NoError(err)
	s.Require().NotNil(found)
	s.Require().Equal("deploy-notes", found.Name)
}

func (s *SqliteSuite) TestUpdateFile_SetAndClearName() {
	database := s.getTestDB(true)

	file := s.createFile(database, "")

	found, err := database.FindFile(context.Background(), file.ID)
	s.Require().NoError(err)
	s.Require().Empty(found.Name)

	file.Name = "deploy-notes"
	s.Require().NoError(database.UpdateFile(context.Background(), file))

	found, err = database.FindFile(context.Background(), file.ID)
	s.Require().NoError(err)
	s.Require().Equal("deploy-notes", found.Name)

	file.Name = ""
	s.Require().NoError(database.UpdateFile(context.Background(), file))

	found, err = database.FindFile(context.Background(), file.ID)
	s.Require().NoError(err)
	s.Require().Empty(found.Name)
}

func (s *SqliteSuite) TestNames_NotUniqueAcrossUsers() {
	database := s.getTestDB(true)

	// different users can reuse a name — the file ID is what disambiguates
	first := s.createFile(database, "deploy-notes")
	second := s.createFile(database, "deploy-notes")

	s.Require().NotEqual(first.ID, second.ID)
	s.Require().NotEqual(first.UserID, second.UserID)

	for _, f := range []*snips.File{first, second} {
		found, err := database.FindFile(context.Background(), f.ID)
		s.Require().NoError(err)
		s.Require().Equal("deploy-notes", found.Name)
	}
}

func (s *SqliteSuite) TestNames_UniquePerUser() {
	database := s.getTestDB(true)

	first := s.createFile(database, "deploy-notes")

	duplicate := &snips.File{
		Size:       11,
		RawContent: []byte("hello world"),
		Type:       "plaintext",
		UserID:     first.UserID,
		Name:       "Deploy-Notes", // same name, different case
	}
	err := database.CreateFile(context.Background(), duplicate, 0)
	s.Require().ErrorIs(err, db.ErrNameTaken)

	// renaming another file to a taken name fails too
	other := &snips.File{
		Size:       11,
		RawContent: []byte("hello world"),
		Type:       "plaintext",
		UserID:     first.UserID,
	}
	s.Require().NoError(database.CreateFile(context.Background(), other, 0))

	other.Name = "DEPLOY-NOTES"
	err = database.UpdateFile(context.Background(), other)
	s.Require().ErrorIs(err, db.ErrNameTaken)

	// updating a file without changing its own name is fine
	first.Size = 22
	s.Require().NoError(database.UpdateFile(context.Background(), first))

	// unnamed files don't participate in the index: many per user are fine
	for range 2 {
		unnamed := &snips.File{
			Size:       11,
			RawContent: []byte("hello world"),
			Type:       "plaintext",
			UserID:     first.UserID,
		}
		s.Require().NoError(database.CreateFile(context.Background(), unnamed, 0))
	}
}

func (s *SqliteSuite) TestFindFilesByUser_IncludesName() {
	database := s.getTestDB(true)

	file := s.createFile(database, "deploy-notes")

	files, err := database.FindFilesByUser(context.Background(), file.UserID)
	s.Require().NoError(err)
	s.Require().Len(files, 1)
	s.Require().Equal("deploy-notes", files[0].Name)
}

func (s *SqliteSuite) TestFindFileByName() {
	database := s.getTestDB(true)

	first := s.createFile(database, "Deploy-Notes")
	s.createFile(database, "other")

	// another user's file with the same name is out of scope
	s.createFile(database, "deploy-notes")

	// case-insensitive, scoped to the user, includes content
	file, err := database.FindFileByName(context.Background(), first.UserID, "DEPLOY-NOTES")
	s.Require().NoError(err)
	s.Require().NotNil(file)
	s.Require().Equal(first.ID, file.ID)
	s.Require().Equal(first.RawContent, file.RawContent)

	file, err = database.FindFileByName(context.Background(), first.UserID, "nope")
	s.Require().NoError(err)
	s.Require().Nil(file)
}

func (s *SqliteSuite) insertFileForPaging(database *db.Sqlite, userID string, updatedAt time.Time, fileID string) *snips.File {
	file := &snips.File{
		ID:        fileID,
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
		Size:      1,
		Private:   false,
		Type:      "plaintext",
		UserID:    userID,
	}

	const query = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.testDB.Exec(query, file.ID, file.CreatedAt, file.UpdatedAt, file.Size, []byte("x"), file.Private, file.Type, file.UserID)
	s.Require().NoError(err)

	return file
}

func (s *SqliteSuite) TestFindFilesByUser_Pagination() {
	database := s.getTestDB(true)
	userID := id.New()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		s.insertFileForPaging(database, userID, base.Add(time.Duration(i)*time.Minute), fmt.Sprintf("file-%d", i))
	}
	// another user's files must never appear
	s.insertFileForPaging(database, id.New(), base.Add(time.Hour), "other-user-file")

	// first page: newest first
	page, err := database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2))
	s.Require().NoError(err)
	s.Require().Len(page, 2)
	s.Require().Equal("file-4", page[0].ID)
	s.Require().Equal("file-3", page[1].ID)

	// second page continues at the offset
	page, err = database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2), db.WithOffset(2))
	s.Require().NoError(err)
	s.Require().Len(page, 2)
	s.Require().Equal("file-2", page[0].ID)
	s.Require().Equal("file-1", page[1].ID)

	// final partial page
	page, err = database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2), db.WithOffset(4))
	s.Require().NoError(err)
	s.Require().Len(page, 1)
	s.Require().Equal("file-0", page[0].ID)

	// past the end
	page, err = database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2), db.WithOffset(5))
	s.Require().NoError(err)
	s.Require().Empty(page)
}

func (s *SqliteSuite) TestFindFilesByUser_PaginationTieBreaksOnID() {
	database := s.getTestDB(true)
	userID := id.New()

	// same updated_at for every file: ordering falls back to id DESC
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, fileID := range []string{"aaa", "bbb", "ccc"} {
		s.insertFileForPaging(database, userID, ts, fileID)
	}

	page, err := database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2))
	s.Require().NoError(err)
	s.Require().Len(page, 2)
	s.Require().Equal("ccc", page[0].ID)
	s.Require().Equal("bbb", page[1].ID)

	page, err = database.FindFilesByUser(context.TODO(), userID, db.WithLimit(2), db.WithOffset(2))
	s.Require().NoError(err)
	s.Require().Len(page, 1)
	s.Require().Equal("aaa", page[0].ID)
}

func (s *SqliteSuite) TestFindRevisionsByFileID_Pagination() {
	database := s.getTestDB(true)
	fileID := id.New()

	for i := 0; i < 5; i++ {
		rev := &snips.Revision{
			FileID:  fileID,
			RawDiff: []byte("diff"),
			Size:    4,
			Type:    "plaintext",
		}
		s.Require().NoError(database.CreateRevision(context.TODO(), rev, 0))
	}

	page, err := database.FindRevisionsByFileID(context.TODO(), fileID, db.WithLimit(2))
	s.Require().NoError(err)
	s.Require().Len(page, 2)
	s.Require().Equal(int64(5), page[0].Sequence)
	s.Require().Equal(int64(4), page[1].Sequence)

	page, err = database.FindRevisionsByFileID(context.TODO(), fileID, db.WithLimit(2), db.WithOffset(2))
	s.Require().NoError(err)
	s.Require().Len(page, 2)
	s.Require().Equal(int64(3), page[0].Sequence)
	s.Require().Equal(int64(2), page[1].Sequence)

	page, err = database.FindRevisionsByFileID(context.TODO(), fileID, db.WithLimit(2), db.WithOffset(4))
	s.Require().NoError(err)
	s.Require().Len(page, 1)
	s.Require().Equal(int64(1), page[0].Sequence)
}
