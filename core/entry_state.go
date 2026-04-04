package core

import (
	"context"
	"fmt"
)

// EntryStateManager owns read/star mutations for entries.
type EntryStateManager struct {
	store EntryStateStore
}

// NewEntryStateManager creates an entry state service.
func NewEntryStateManager(store EntryStateStore) *EntryStateManager {
	return &EntryStateManager{store: store}
}

func runEntryStateAction(ctx context.Context, id int64, label string, action func(context.Context, int64) error) error {
	if err := action(ctx, id); err != nil {
		return fmt.Errorf("%s %d: %w", label, id, err)
	}
	return nil
}

func runEntryStateBatchAction(ctx context.Context, ids []int64, label string, action func(context.Context, []int64) error) error {
	if len(ids) == 0 {
		return nil
	}
	if err := action(ctx, ids); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

// MarkEntryRead marks an entry as read.
func (m *EntryStateManager) MarkEntryRead(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "mark read", m.store.MarkEntryRead)
}

// MarkEntriesRead marks multiple entries as read.
func (m *EntryStateManager) MarkEntriesRead(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "mark read entries", m.store.MarkEntriesRead)
}

// MarkEntryUnread marks an entry as unread.
func (m *EntryStateManager) MarkEntryUnread(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "mark unread", m.store.MarkEntryUnread)
}

// MarkEntriesUnread marks multiple entries as unread.
func (m *EntryStateManager) MarkEntriesUnread(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "mark unread entries", m.store.MarkEntriesUnread)
}

// StarEntry bookmarks an entry.
func (m *EntryStateManager) StarEntry(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "star entry", m.store.StarEntry)
}

// StarEntries bookmarks multiple entries.
func (m *EntryStateManager) StarEntries(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "star entries", m.store.StarEntries)
}

// UnstarEntry removes a bookmark from an entry.
func (m *EntryStateManager) UnstarEntry(ctx context.Context, id int64) error {
	return runEntryStateAction(ctx, id, "unstar entry", m.store.UnstarEntry)
}

// UnstarEntries removes bookmarks from multiple entries.
func (m *EntryStateManager) UnstarEntries(ctx context.Context, ids []int64) error {
	return runEntryStateBatchAction(ctx, ids, "unstar entries", m.store.UnstarEntries)
}

func (s *Service) MarkEntryRead(ctx context.Context, id int64) error {
	return s.entryState.MarkEntryRead(ctx, id)
}

func (s *Service) MarkEntriesRead(ctx context.Context, ids []int64) error {
	return s.entryState.MarkEntriesRead(ctx, ids)
}

func (s *Service) MarkEntryUnread(ctx context.Context, id int64) error {
	return s.entryState.MarkEntryUnread(ctx, id)
}

func (s *Service) MarkEntriesUnread(ctx context.Context, ids []int64) error {
	return s.entryState.MarkEntriesUnread(ctx, ids)
}

func (s *Service) StarEntry(ctx context.Context, id int64) error {
	return s.entryState.StarEntry(ctx, id)
}

func (s *Service) StarEntries(ctx context.Context, ids []int64) error {
	return s.entryState.StarEntries(ctx, ids)
}

func (s *Service) UnstarEntry(ctx context.Context, id int64) error {
	return s.entryState.UnstarEntry(ctx, id)
}

func (s *Service) UnstarEntries(ctx context.Context, ids []int64) error {
	return s.entryState.UnstarEntries(ctx, ids)
}
