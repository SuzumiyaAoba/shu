package core

import "context"

// FeedStatsAll returns aggregate statistics for all feeds.
func (s *Service) FeedStatsAll(ctx context.Context) ([]FeedStats, error) {
	return s.store.FeedStats(ctx)
}
