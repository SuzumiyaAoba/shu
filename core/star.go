package core

import (
	"context"
)

// StarEntry bookmarks an entry.
func (s *Service) StarEntry(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "star entry %d: %w", s.store.StarEntry)
}

// StarEntries bookmarks multiple entries.
func (s *Service) StarEntries(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "star entries: %w", s.store.StarEntries)
}

// UnstarEntry removes a bookmark from an entry.
func (s *Service) UnstarEntry(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "unstar entry %d: %w", s.store.UnstarEntry)
}

// UnstarEntries removes bookmarks from multiple entries.
func (s *Service) UnstarEntries(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "unstar entries: %w", s.store.UnstarEntries)
}
