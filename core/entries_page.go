package core

import "context"

func buildEntryPage(entries []*Entry, total, offset, limit int) *EntryPage {
	if limit == 0 {
		limit = len(entries)
	}

	return &EntryPage{
		Entries:    entries,
		TotalCount: total,
		Offset:     offset,
		Limit:      limit,
		HasMore:    offset+len(entries) < total,
	}
}

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

	return buildEntryPage(entries, total, filter.Offset, filter.Limit), nil
}
