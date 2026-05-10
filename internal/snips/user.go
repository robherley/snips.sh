package snips

import "time"

type User struct {
	ID         string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ThemeColor string // hex color (e.g. "#65adff"); empty = use default palette
}
