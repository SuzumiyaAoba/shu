package core

import "context"

// SearchEntries performs full-text search across entry titles, summaries, and
// content using the FTS5 index.
func (s *Service) SearchEntries(ctx context.Context, query string, limit int) ([]*Entry, error) {
	return s.store.SearchEntries(ctx, query, limit)
}
