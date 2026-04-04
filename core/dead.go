package core

import "context"

// ListDeadFeeds returns feeds with recorded fetch failures.
// Manually disabled feeds without recorded errors are excluded.
func (m *MaintenanceOps) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
	feeds, err := m.feedStore.ListFeeds(ctx)
	if err != nil {
		return nil, err
	}

	dead := make([]*Feed, 0)
	for _, feed := range feeds {
		if feed.ErrorCount > 0 {
			dead = append(dead, feed)
		}
	}
	return dead, nil
}

// RemoveDeadFeeds deletes feeds with recorded fetch failures and returns the
// removed feeds.
func (m *MaintenanceOps) RemoveDeadFeeds(ctx context.Context) ([]*Feed, error) {
	dead, err := m.ListDeadFeeds(ctx)
	if err != nil {
		return nil, err
	}

	for _, feed := range dead {
		if err := m.feedStore.RemoveFeed(ctx, feed.ID); err != nil {
			return nil, err
		}
		m.logger.Info("feed removed", "id", feed.ID)
	}
	return dead, nil
}

func (s *Service) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
	return s.maintenance.ListDeadFeeds(ctx)
}

func (s *Service) RemoveDeadFeeds(ctx context.Context) ([]*Feed, error) {
	return s.maintenance.RemoveDeadFeeds(ctx)
}
