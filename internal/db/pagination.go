package db

// Cursor contains backend-neutral pagination state. SQLite uses Offset while
// PostgreSQL uses the seek fields appropriate to each listing query.
type Cursor struct {
	Offset uint64
	ID     string
}

// Page contains resolved pagination settings for a database backend.
type Page struct {
	Limit  uint64
	Offset uint64
	Cursor Cursor
}

// PageOption tunes pagination on listing queries; omitting options returns
// all rows.
type PageOption func(*Page)

// WithLimit caps the number of rows returned.
func WithLimit(limit uint64) PageOption {
	return func(page *Page) { page.Limit = limit }
}

// WithCursor resumes a listing from an opaque API cursor.
func WithCursor(cursor Cursor) PageOption {
	return func(page *Page) {
		page.Cursor = cursor
		page.Offset = cursor.Offset
	}
}

// ResolvePage applies pagination options for use by a database backend.
func ResolvePage(opts ...PageOption) Page {
	page := Page{}
	for _, opt := range opts {
		opt(&page)
	}
	return page
}
