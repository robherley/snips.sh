package db_test

import (
	"context"
	"database/sql"
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

func (s *SqliteSuite) TestLatestPublicFiles() {
	database := s.getTestDB(true)

	existingFiles := []*snips.File{
		{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world"),
			Private:    false,
			Type:       "plaintext",
			UserID:     id.New(),
		},
		{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world, again"),
			Private:    false,
			Type:       "plaintext",
			UserID:     id.New(),
		},
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

	for _, existingFile := range existingFiles {
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

	filesPageOne, err := database.LatestPublicFiles(context.TODO(), 1, 1)

	// raw content is not returned in LatestPublicFiles
	expected := existingFiles[1]
	expected.RawContent = nil

	s.Require().NoError(err)
	s.Require().Equal(expected, filesPageOne[0])

	filesPageTwo, err := database.LatestPublicFiles(context.TODO(), 2, 1)

	// raw content is not returned in LatestPublicFiles
	expected = existingFiles[0]
	expected.RawContent = nil

	s.Require().NoError(err)
	s.Require().Equal(existingFiles[0], filesPageTwo[0])
}

func (s *SqliteSuite) TestLatestPublicFiles_PublicOnly() {
	database := s.getTestDB(true)

	existingFiles := []*snips.File{
		{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world"),
			Private:    false,
			Type:       "plaintext",
			UserID:     id.New(),
		},
		{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world, I am private"),
			Private:    true,
			Type:       "plaintext",
			UserID:     id.New(),
		},
		{
			ID:         id.New(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			Size:       11,
			RawContent: []byte("hello world, again"),
			Private:    false,
			Type:       "plaintext",
			UserID:     id.New(),
		},
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

	for _, existingFile := range existingFiles {
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

	filesPageOne, err := database.LatestPublicFiles(context.TODO(), 1, 1)

	// raw content is not returned in LatestPublicFiles
	expected := existingFiles[2]
	expected.RawContent = nil

	s.Require().NoError(err)
	s.Require().Equal(expected, filesPageOne[0])

	filesPageTwo, err := database.LatestPublicFiles(context.TODO(), 2, 1)

	// raw content is not returned in LatestPublicFiles
	expected = existingFiles[0]
	expected.RawContent = nil

	s.Require().NoError(err)
	s.Require().Equal(expected, filesPageTwo[0])
}

func (s *SqliteSuite) TestLatestPublicFiles_EmptyResults() {
	database := s.getTestDB(true)

	filesPageOne, err := database.LatestPublicFiles(context.TODO(), 2, 1)
	s.Require().NoError(err)
	s.Require().Len(filesPageOne, 0)
}
