package core

import "context"

// FindDuplicateEntries returns entries from other feeds that share the same
// link URL as the given entry.
func (s *Service) FindDuplicateEntries(ctx context.Context, entryID int64) ([]*Entry, error) {
	return s.store.FindDuplicateEntries(ctx, entryID)
}
