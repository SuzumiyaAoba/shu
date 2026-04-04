package core

import (
	"context"
	"fmt"
)

func runEntryStateAction(ctx context.Context, id int64, format string, action func(context.Context, int64) error) error {
	if err := action(ctx, id); err != nil {
		return fmt.Errorf(format, id, err)
	}
	return nil
}

func runEntryStateBatchAction(ctx context.Context, ids []int64, format string, action func(context.Context, []int64) error) error {
	if len(ids) == 0 {
		return nil
	}
	if err := action(ctx, ids); err != nil {
		return fmt.Errorf(format, err)
	}
	return nil
}

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
