package core

import "context"

// ListEntriesPage returns entries plus pagination metadata for the given filter.
func (s *Service) ListEntriesPage(ctx context.Context, filter EntryFilter) (*EntryPage, error) {
	entries, err := s.store.ListEntries(ctx, filter)
	if err != nil {
		return nil, err
	}

	total, err := s.store.CountEntries(ctx, filter)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = len(entries)
	}

	return &EntryPage{
		Entries:    entries,
		TotalCount: total,
		Offset:     filter.Offset,
		Limit:      limit,
		HasMore:    filter.Offset+len(entries) < total,
	}, nil
}
