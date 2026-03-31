package core

import (
	"context"
	"fmt"
	"time"
)

// CleanupEntries deletes entries older than the given duration, excluding
// starred entries. Returns the number of deleted entries.
func (s *Service) CleanupEntries(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	n, err := s.store.CleanupEntries(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup: %w", err)
	}
	s.logger.Info("cleanup completed", "deleted", n)
	return n, nil
}
