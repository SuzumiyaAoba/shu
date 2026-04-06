package core

import (
	"context"

	"github.com/SuzumiyaAoba/shu/model"
)

// EntryQueries owns read-only entry lookup and search flows.
type EntryQueries struct {
	store EntryStore
}

// NewEntryQueries creates an entry query service.
func NewEntryQueries(store EntryStore) *EntryQueries {
	return &EntryQueries{store: store}
}

func buildEntryPage(entries []*model.Entry, total, offset, limit int) *model.EntryPage {
	if limit == 0 {
		limit = len(entries)
	}

	return &model.EntryPage{
		Entries:    entries,
		TotalCount: total,
		Offset:     offset,
		Limit:      limit,
		HasMore:    offset+len(entries) < total,
	}
}

// GetEntry retrieves a single entry by its primary key.
func (q *EntryQueries) GetEntry(ctx context.Context, id int64) (*model.Entry, error) {
	return q.store.GetEntry(ctx, id)
}

// ListEntries retrieves stored entries matching the given filter criteria.
// Results are ordered by fetched_at descending (newest first). It delegates
// directly to the store without additional business logic.
func (q *EntryQueries) ListEntries(ctx context.Context, filter model.EntryFilter) ([]*model.Entry, error) {
	return q.store.ListEntries(ctx, filter)
}

// ListEntriesPage returns entries plus pagination metadata for the given filter.
func (q *EntryQueries) ListEntriesPage(ctx context.Context, filter model.EntryFilter) (*model.EntryPage, error) {
	entries, err := q.store.ListEntries(ctx, filter)
	if err != nil {
		return nil, err
	}

	total, err := q.store.CountEntries(ctx, filter)
	if err != nil {
		return nil, err
	}

	return buildEntryPage(entries, total, filter.Offset, filter.Limit), nil
}

func (s *Service) GetEntry(ctx context.Context, id int64) (*model.Entry, error) {
	return s.entries.GetEntry(ctx, id)
}

func (s *Service) ListEntries(ctx context.Context, filter model.EntryFilter) ([]*model.Entry, error) {
	return s.entries.ListEntries(ctx, filter)
}

func (s *Service) ListEntriesPage(ctx context.Context, filter model.EntryFilter) (*model.EntryPage, error) {
	return s.entries.ListEntriesPage(ctx, filter)
}
