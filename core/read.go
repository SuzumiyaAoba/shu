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

// MarkEntryUnread marks an entry as unread.
func (s *Service) MarkEntryUnread(ctx context.Context, id int64) error {
	if err := s.store.MarkEntryUnread(ctx, id); err != nil {
		return fmt.Errorf("mark unread %d: %w", id, err)
	}
	return nil
}
