package snips

import "time"

type User struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ThemeColor string    `json:"-"`
}
