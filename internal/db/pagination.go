package db

// page collects the effects of PageOptions on a listing query.
type page struct {
	limit  uint64
	offset uint64
}

// PageOption tunes pagination on listing queries; omitting options returns
// all rows.
type PageOption func(*page)

// WithLimit caps the number of rows returned.
func WithLimit(limit uint64) PageOption {
	return func(p *page) { p.limit = limit }
}

// WithOffset skips rows before the returned page.
func WithOffset(offset uint64) PageOption {
	return func(p *page) { p.offset = offset }
}

func buildPage(opts []PageOption) page {
	p := page{}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}
