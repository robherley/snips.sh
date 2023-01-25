package db

type File struct {
	Model

	Size      uint64
	Content   []byte
	Private   bool `gorm:"index:file_private"`
	Extension *string

	UserID string
	User   User
}
