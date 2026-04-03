package core

import (
	"context"
)

// MarkEntryRead marks an entry as read.
func (s *Service) MarkEntryRead(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "mark read %d: %w", s.store.MarkEntryRead)
}

// MarkEntriesRead marks multiple entries as read.
func (s *Service) MarkEntriesRead(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "mark read entries: %w", s.store.MarkEntriesRead)
}

// MarkEntryUnread marks an entry as unread.
func (s *Service) MarkEntryUnread(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "mark unread %d: %w", s.store.MarkEntryUnread)
}

// MarkEntriesUnread marks multiple entries as unread.
func (s *Service) MarkEntriesUnread(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "mark unread entries: %w", s.store.MarkEntriesUnread)
}
