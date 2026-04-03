package core

import "context"

// ListDeadFeeds returns feeds that are disabled due to fetch failures.
// Manually disabled feeds without recorded errors are excluded.
func (s *Service) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		return nil, err
	}

	dead := make([]*Feed, 0)
	for _, feed := range feeds {
		if feed.Disabled && feed.ErrorCount > 0 {
			dead = append(dead, feed)
		}
	}
	return dead, nil
}

// RemoveDeadFeeds deletes feeds that are disabled due to fetch failures and
// returns the removed feeds.
func (s *Service) RemoveDeadFeeds(ctx context.Context) ([]*Feed, error) {
	dead, err := s.ListDeadFeeds(ctx)
	if err != nil {
		return nil, err
	}

	for _, feed := range dead {
		if err := s.RemoveFeed(ctx, feed.ID); err != nil {
			return nil, err
		}
	}
	return dead, nil
}
