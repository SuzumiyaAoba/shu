package core

import (
	"context"

	"github.com/SuzumiyaAoba/shu/model"
)

// FindDuplicateEntries returns entries from other feeds that share the same
// link URL as the given entry.
func (q *EntryQueries) FindDuplicateEntries(ctx context.Context, entryID int64) ([]*model.Entry, error) {
	return q.store.FindDuplicateEntries(ctx, entryID)
}

func (s *Service) FindDuplicateEntries(ctx context.Context, entryID int64) ([]*model.Entry, error) {
	return s.entries.FindDuplicateEntries(ctx, entryID)
}
