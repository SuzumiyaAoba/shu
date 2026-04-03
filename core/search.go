package core

import "context"

// SearchEntries performs full-text search across entry titles, summaries, and
// content using the FTS5 index.
func (s *Service) SearchEntries(ctx context.Context, query string, limit int) ([]*Entry, error) {
	return s.store.SearchEntries(ctx, query, limit)
}

// SearchEntriesPage returns paginated full-text search results plus metadata.
func (s *Service) SearchEntriesPage(ctx context.Context, query string, limit, offset int) (*EntryPage, error) {
	entries, err := s.store.SearchEntriesPage(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.store.CountSearchEntries(ctx, query)
	if err != nil {
		return nil, err
	}

	return buildEntryPage(entries, total, offset, limit), nil
}
