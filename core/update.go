package core

import (
	"context"
	"fmt"

	"github.com/SuzumiyaAoba/shu/model"
)

// UpdateFeed modifies mutable fields of an existing feed.
func (m *FeedManager) UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error {
	if err := m.store.UpdateFeed(ctx, id, update); err != nil {
		return fmt.Errorf("update feed %d: %w", id, err)
	}
	m.logger.Info("feed updated", "id", id)
	return nil
}

// EnableFeed re-enables a disabled feed and resets its error count.
func (m *FeedManager) EnableFeed(ctx context.Context, id int64) error {
	if err := m.store.RunInTx(ctx, func(ctx context.Context) error {
		if err := m.store.SetFeedDisabled(ctx, id, false); err != nil {
			return fmt.Errorf("enable feed %d: %w", id, err)
		}
		if err := m.store.ResetFeedError(ctx, id); err != nil {
			return fmt.Errorf("reset errors feed %d: %w", id, err)
		}
		return nil
	}); err != nil {
		return err
	}
	m.logger.Info("feed enabled", "id", id)
	return nil
}

// DisableFeed disables a feed so it is skipped during fetch.
func (m *FeedManager) DisableFeed(ctx context.Context, id int64) error {
	if err := m.store.SetFeedDisabled(ctx, id, true); err != nil {
		return fmt.Errorf("disable feed %d: %w", id, err)
	}
	m.logger.Info("feed disabled", "id", id)
	return nil
}

func (s *Service) UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error {
	return s.feeds.UpdateFeed(ctx, id, update)
}

func (s *Service) EnableFeed(ctx context.Context, id int64) error {
	return s.feeds.EnableFeed(ctx, id)
}

func (s *Service) DisableFeed(ctx context.Context, id int64) error {
	return s.feeds.DisableFeed(ctx, id)
}
