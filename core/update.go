package core

import (
	"context"
	"fmt"
)

// UpdateFeed modifies mutable fields of an existing feed.
func (s *Service) UpdateFeed(ctx context.Context, id int64, update FeedUpdate) error {
	if err := s.store.UpdateFeed(ctx, id, update); err != nil {
		return fmt.Errorf("update feed %d: %w", id, err)
	}
	s.logger.Info("feed updated", "id", id)
	return nil
}
