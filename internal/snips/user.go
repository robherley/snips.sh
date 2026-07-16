package snips

import "time"

type User struct {
	ID         string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ThemeColor string // named theme option from styles.ThemeOptions (e.g. "blue"); empty = use default palette
}
