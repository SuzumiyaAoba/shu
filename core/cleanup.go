package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type maintenanceFeedStore interface {
	FeedStore
	TxRunner
}

// MaintenanceOps owns cleanup and maintenance workflows.
type MaintenanceOps struct {
	feedStore maintenanceFeedStore
	store     MaintenanceStore
	logger    *slog.Logger
}

// NewMaintenanceOps creates a maintenance domain service.
func NewMaintenanceOps(feedStore maintenanceFeedStore, store MaintenanceStore, logger *slog.Logger) *MaintenanceOps {
	return &MaintenanceOps{
		feedStore: feedStore,
		store:     store,
		logger:    normalizeLogger(logger),
	}
}

// CleanupEntries deletes entries older than the given duration, excluding
// starred entries. Returns the number of deleted entries.
func (m *MaintenanceOps) CleanupEntries(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	n, err := m.store.CleanupEntries(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup: %w", err)
	}
	m.logger.Info("cleanup completed", "deleted", n)
	return n, nil
}

func (s *Service) CleanupEntries(ctx context.Context, olderThan time.Duration) (int, error) {
	return s.maintenance.CleanupEntries(ctx, olderThan)
}
