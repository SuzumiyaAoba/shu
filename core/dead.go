package core

import "context"

// ListDeadFeeds returns feeds with recorded fetch failures.
// Manually disabled feeds without recorded errors are excluded.
func (m *MaintenanceOps) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
	return m.feedStore.ListDeadFeeds(ctx)
}

// RemoveDeadFeeds deletes feeds with recorded fetch failures and returns the
// removed feeds.
func (m *MaintenanceOps) RemoveDeadFeeds(ctx context.Context) ([]*Feed, error) {
	dead, err := m.ListDeadFeeds(ctx)
	if err != nil {
		return nil, err
	}

	if err := m.feedStore.RunInTx(ctx, func(ctx context.Context) error {
		for _, feed := range dead {
			if err := m.feedStore.RemoveFeed(ctx, feed.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for _, feed := range dead {
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
