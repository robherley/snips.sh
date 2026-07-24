package db

// Page contains resolved pagination settings for a database backend.
type Page struct {
	Limit  uint64
	Offset uint64
}

// PageOption tunes pagination on listing queries; omitting options returns
// all rows.
type PageOption func(*Page)

// WithLimit caps the number of rows returned.
func WithLimit(limit uint64) PageOption {
	return func(page *Page) { page.Limit = limit }
}

// WithOffset skips rows before the returned page.
func WithOffset(offset uint64) PageOption {
	return func(page *Page) { page.Offset = offset }
}

// ResolvePage applies pagination options for use by a database backend.
func ResolvePage(opts ...PageOption) Page {
	page := Page{}
	for _, opt := range opts {
		opt(&page)
	}
	return page
}
