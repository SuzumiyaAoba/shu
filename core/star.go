package core

import (
	"context"
	"fmt"
)

// StarEntry bookmarks an entry.
func (s *Service) StarEntry(ctx context.Context, id int64) error {
	if err := s.store.StarEntry(ctx, id); err != nil {
		return fmt.Errorf("star entry %d: %w", id, err)
	}
	return nil
}

// StarEntries bookmarks multiple entries.
func (s *Service) StarEntries(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.store.StarEntries(ctx, ids); err != nil {
		return fmt.Errorf("star entries: %w", err)
	}
	return nil
}

// UnstarEntry removes a bookmark from an entry.
func (s *Service) UnstarEntry(ctx context.Context, id int64) error {
	if err := s.store.UnstarEntry(ctx, id); err != nil {
		return fmt.Errorf("unstar entry %d: %w", id, err)
	}
	return nil
}

// UnstarEntries removes bookmarks from multiple entries.
func (s *Service) UnstarEntries(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.store.UnstarEntries(ctx, ids); err != nil {
		return fmt.Errorf("unstar entries: %w", err)
	}
	return nil
}
