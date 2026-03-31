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

// EnableFeed re-enables a disabled feed and resets its error count.
func (s *Service) EnableFeed(ctx context.Context, id int64) error {
	if err := s.store.SetFeedDisabled(ctx, id, false); err != nil {
		return fmt.Errorf("enable feed %d: %w", id, err)
	}
	if err := s.store.ResetFeedError(ctx, id); err != nil {
		return fmt.Errorf("reset errors feed %d: %w", id, err)
	}
	s.logger.Info("feed enabled", "id", id)
	return nil
}

// DisableFeed disables a feed so it is skipped during fetch.
func (s *Service) DisableFeed(ctx context.Context, id int64) error {
	if err := s.store.SetFeedDisabled(ctx, id, true); err != nil {
		return fmt.Errorf("disable feed %d: %w", id, err)
	}
	s.logger.Info("feed disabled", "id", id)
	return nil
}
