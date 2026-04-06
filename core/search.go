package core

import (
	"context"

	"github.com/SuzumiyaAoba/shu/model"
)

// SearchEntries performs full-text search across entry titles, summaries, and
// content using the FTS5 index.
func (q *EntryQueries) SearchEntries(ctx context.Context, query string, limit int) ([]*model.Entry, error) {
	return q.store.SearchEntries(ctx, query, limit)
}

// SearchEntriesPage returns paginated full-text search results plus metadata.
func (q *EntryQueries) SearchEntriesPage(ctx context.Context, query string, limit, offset int) (*model.EntryPage, error) {
	entries, err := q.store.SearchEntriesPage(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := q.store.CountSearchEntries(ctx, query)
	if err != nil {
		return nil, err
	}

	return buildEntryPage(entries, total, offset, limit), nil
}

func (s *Service) SearchEntries(ctx context.Context, query string, limit int) ([]*model.Entry, error) {
	return s.entries.SearchEntries(ctx, query, limit)
}

func (s *Service) SearchEntriesPage(ctx context.Context, query string, limit, offset int) (*model.EntryPage, error) {
	return s.entries.SearchEntriesPage(ctx, query, limit, offset)
}
