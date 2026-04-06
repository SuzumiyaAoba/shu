package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SuzumiyaAoba/shu/model"
	sqlite "modernc.org/sqlite"
)

const sqliteConstraintUnique = 2067

// AddFeed inserts a new feed row. On success the feed's ID is set to the
// auto-generated primary key and AddedAt is set to the current UTC time.
// Returns an error if a feed with the same URL already exists (UNIQUE
// constraint violation).
func (s *SQLiteStore) AddFeed(ctx context.Context, feed *model.Feed) error {
	now := time.Now().UTC()
	result, err := s.executor(ctx).ExecContext(ctx,
		`INSERT INTO feeds (url, title, site_url, added_at, description, language, image_url, feed_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		feed.URL, feed.Title, feed.SiteURL, nowRFC3339(),
		feed.Description, feed.Language, feed.ImageURL, feed.FeedType,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique {
			return fmt.Errorf("feed %s: %w", feed.URL, model.ErrFeedAlreadyExists)
		}
		return &model.StoreError{Op: "add", Table: "feeds", Err: err}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	feed.ID = id
	feed.AddedAt = now
	return nil
}

// GetFeed retrieves a single feed by its primary key.
// Returns a "scan feed" error wrapping sql.ErrNoRows if the ID does not exist.
func (s *SQLiteStore) GetFeed(ctx context.Context, id int64) (*model.Feed, error) {
	row := s.executor(ctx).QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE id = ?`, id,
	)
	return fetchFeed(row, fmt.Sprintf("feed %d", id))
}

// GetFeedByURL retrieves a single feed by its unique URL.
func (s *SQLiteStore) GetFeedByURL(ctx context.Context, url string) (*model.Feed, error) {
	row := s.executor(ctx).QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE url = ?`, url,
	)
	return fetchFeed(row, fmt.Sprintf("feed %s", url))
}

// ListFeeds returns all registered feeds ordered by ascending ID.
// Returns an empty (non-nil) slice if no feeds are registered.
func (s *SQLiteStore) ListFeeds(ctx context.Context) ([]*model.Feed, error) {
	rows, err := s.executor(ctx).QueryContext(ctx,
		`SELECT `+feedColumns+` FROM feeds ORDER BY id`,
	)
	if err != nil {
		return nil, &model.StoreError{Op: "list", Table: "feeds", Err: err}
	}
	return collectFeeds(rows)
}

// RemoveFeed deletes the feed with the given ID. Due to the ON DELETE CASCADE
// constraint on the entries table, all entries belonging to this feed are also
// deleted. Returns [model.ErrFeedNotFound] if no feed with that ID exists.
func (s *SQLiteStore) RemoveFeed(ctx context.Context, id int64) error {
	result, err := s.executor(ctx).ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("feed %d: %w", id, model.ErrFeedNotFound)
	}
	return nil
}

// UpdateFeedFetchedAt sets the feed's fetched_at column to the current UTC
// time. This is called after a successful fetch cycle to record when the feed
// was last refreshed.
func (s *SQLiteStore) UpdateFeedFetchedAt(ctx context.Context, id int64) error {
	_, err := s.executor(ctx).ExecContext(ctx,
		`UPDATE feeds SET fetched_at = ? WHERE id = ?`, nowRFC3339(), id,
	)
	if err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

// UpdateFeed updates mutable feed fields. Only non-nil fields in the update
// struct are applied.
func (s *SQLiteStore) UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error {
	var sets []string
	var args []any

	if update.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *update.Title)
	}
	if update.URL != nil {
		sets = append(sets, "url = ?")
		args = append(args, *update.URL)
	}
	if update.FetchIntervalSec != nil {
		sets = append(sets, "fetch_interval_sec = ?")
		args = append(args, *update.FetchIntervalSec)
	}

	if len(sets) == 0 {
		return nil
	}

	args = append(args, id)
	query := "UPDATE feeds SET " + strings.Join(sets, ", ") + " WHERE id = ?"

	result, err := s.executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("feed %d: %w", id, model.ErrFeedNotFound)
	}
	return nil
}

// UpdateFeedCacheHeaders stores the HTTP ETag and Last-Modified headers
// received during a fetch.
func (s *SQLiteStore) UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error {
	_, err := s.executor(ctx).ExecContext(ctx,
		`UPDATE feeds SET etag = ?, last_modified = ? WHERE id = ?`,
		etag, lastModified, id,
	)
	if err != nil {
		return fmt.Errorf("update cache headers: %w", err)
	}
	return nil
}

// ListDeadFeeds returns feeds that have at least one recorded fetch error.
// Filtering is done in SQL to avoid loading all feeds into memory.
func (s *SQLiteStore) ListDeadFeeds(ctx context.Context) ([]*model.Feed, error) {
	rows, err := s.executor(ctx).QueryContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE error_count > 0 ORDER BY id`,
	)
	if err != nil {
		return nil, &model.StoreError{Op: "list_dead", Table: "feeds", Err: err}
	}
	return collectFeeds(rows)
}

// maxErrorCount is the number of consecutive failures before a feed is
// automatically disabled.
const maxErrorCount = 5

// RecordFeedError increments the error count and stores the error message.
// If the count reaches maxErrorCount the feed is automatically disabled.
func (s *SQLiteStore) RecordFeedError(ctx context.Context, id int64, errMsg string) error {
	_, err := s.executor(ctx).ExecContext(ctx,
		`UPDATE feeds SET error_count = error_count + 1, last_error = ?, disabled = CASE WHEN error_count + 1 >= ? THEN 1 ELSE disabled END WHERE id = ?`,
		errMsg, maxErrorCount, id,
	)
	if err != nil {
		return fmt.Errorf("record feed error: %w", err)
	}
	return nil
}

// ResetFeedError clears the error count and last error after a successful fetch.
func (s *SQLiteStore) ResetFeedError(ctx context.Context, id int64) error {
	_, err := s.executor(ctx).ExecContext(ctx,
		`UPDATE feeds SET error_count = 0, last_error = '' WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("reset feed error: %w", err)
	}
	return nil
}

// SetFeedDisabled enables or disables a feed.
func (s *SQLiteStore) SetFeedDisabled(ctx context.Context, id int64, disabled bool) error {
	val := 0
	if disabled {
		val = 1
	}
	_, err := s.executor(ctx).ExecContext(ctx, `UPDATE feeds SET disabled = ? WHERE id = ?`, val, id)
	if err != nil {
		return fmt.Errorf("set feed disabled: %w", err)
	}
	return nil
}
