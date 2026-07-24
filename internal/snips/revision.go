package snips

import "time"

type Revision struct {
	ID        string    `json:"id"`
	Sequence  int64     `json:"sequence"`
	FileID    string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	Size      uint64    `json:"size"` // file size after this revision
	Type      string    `json:"type"` // file type after this revision
}
