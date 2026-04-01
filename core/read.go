package core

import (
	"context"
	"fmt"
)

// MarkEntryRead marks an entry as read.
func (s *Service) MarkEntryRead(ctx context.Context, id int64) error {
	if err := s.store.MarkEntryRead(ctx, id); err != nil {
		return fmt.Errorf("mark read %d: %w", id, err)
	}
	return nil
}

// MarkEntriesRead marks multiple entries as read.
func (s *Service) MarkEntriesRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.store.MarkEntriesRead(ctx, ids); err != nil {
		return fmt.Errorf("mark read entries: %w", err)
	}
	return nil
}

// MarkEntryUnread marks an entry as unread.
func (s *Service) MarkEntryUnread(ctx context.Context, id int64) error {
	if err := s.store.MarkEntryUnread(ctx, id); err != nil {
		return fmt.Errorf("mark unread %d: %w", id, err)
	}
	return nil
}

// MarkEntriesUnread marks multiple entries as unread.
func (s *Service) MarkEntriesUnread(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.store.MarkEntriesUnread(ctx, ids); err != nil {
		return fmt.Errorf("mark unread entries: %w", err)
	}
	return nil
}
