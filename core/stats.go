package core

import "context"

// FeedStatsAll returns aggregate statistics for all feeds.
func (m *MaintenanceOps) FeedStatsAll(ctx context.Context) ([]FeedStats, error) {
	return m.store.FeedStats(ctx)
}

func (s *Service) FeedStatsAll(ctx context.Context) ([]FeedStats, error) {
	return s.maintenance.FeedStatsAll(ctx)
}
