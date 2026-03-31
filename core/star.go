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

// UnstarEntry removes a bookmark from an entry.
func (s *Service) UnstarEntry(ctx context.Context, id int64) error {
	if err := s.store.UnstarEntry(ctx, id); err != nil {
		return fmt.Errorf("unstar entry %d: %w", id, err)
	}
	return nil
}
