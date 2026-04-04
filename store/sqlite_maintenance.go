package store

import (
	"context"
	"fmt"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

// FeedStats returns aggregate statistics for all feeds.
func (s *SQLiteStore) FeedStats(ctx context.Context) ([]core.FeedStats, error) {
	rows, err := s.executor(ctx).QueryContext(ctx, `
		SELECT
			f.id, f.title, f.url,
			COUNT(e.id) AS total,
			SUM(CASE WHEN e.id IS NOT NULL AND e.read_at IS NULL THEN 1 ELSE 0 END) AS unread,
			SUM(CASE WHEN e.starred_at IS NOT NULL THEN 1 ELSE 0 END) AS starred,
			f.fetched_at, f.error_count, f.last_error, f.disabled
		FROM feeds f
		LEFT JOIN entries e ON e.feed_id = f.id
		GROUP BY f.id
		ORDER BY f.id
	`)
	if err != nil {
		return nil, fmt.Errorf("feed stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []core.FeedStats
	for rows.Next() {
		st, err := scanFeedStats(rows)
		if err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

// CleanupEntries deletes entries older than the given time, excluding starred
// entries. Returns the number of deleted entries.
func (s *SQLiteStore) CleanupEntries(ctx context.Context, olderThan time.Time) (int, error) {
	cutoff := olderThan.UTC().Format(time.RFC3339)
	result, err := s.executor(ctx).ExecContext(ctx,
		`DELETE FROM entries WHERE fetched_at < ? AND starred_at IS NULL`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup entries: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func scanFeedStats(s scanner) (core.FeedStats, error) {
	var stats core.FeedStats
	var fetchedAt *string
	var disabled int

	if err := s.Scan(
		&stats.FeedID, &stats.Title, &stats.URL,
		&stats.TotalCount, &stats.UnreadCount, &stats.StarredCount,
		&fetchedAt, &stats.ErrorCount, &stats.LastError, &disabled,
	); err != nil {
		return core.FeedStats{}, fmt.Errorf("scan feed stats: %w", err)
	}

	var err error
	if stats.FetchedAt, err = parseNullableTime(fetchedAt, "fetched_at"); err != nil {
		return core.FeedStats{}, err
	}
	stats.Disabled = disabled != 0

	return stats, nil
}
