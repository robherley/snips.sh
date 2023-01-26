package db

type File struct {
	Model

	Size    uint64
	Content []byte
	Private bool `gorm:"index:file_private"`
	Type    string

	UserID string
	User   User
}

func (f *File) IsBinary() bool {
	return f.Type == "binary"
}
