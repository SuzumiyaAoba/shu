package store

import (
	"context"
	"fmt"

	"github.com/SuzumiyaAoba/shu/core"
)

// AddTag creates a tag (if it doesn't exist) and associates it with a feed.
func (s *SQLiteStore) AddTag(ctx context.Context, feedID int64, tagName string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name) VALUES (?)`, tagName)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO feed_tags (feed_id, tag_id) SELECT ?, id FROM tags WHERE name = ?`,
		feedID, tagName,
	)
	if err != nil {
		return fmt.Errorf("associate tag: %w", err)
	}
	return nil
}

// RemoveTag removes a tag association from a feed.
func (s *SQLiteStore) RemoveTag(ctx context.Context, feedID int64, tagName string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM feed_tags WHERE feed_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`,
		feedID, tagName,
	)
	if err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	return nil
}

// ListTags returns all tags associated with a given feed.
func (s *SQLiteStore) ListTags(ctx context.Context, feedID int64) ([]core.Tag, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.name FROM tags t JOIN feed_tags ft ON ft.tag_id = t.id WHERE ft.feed_id = ? ORDER BY t.name`,
		feedID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return collectTags(rows)
}

// ListAllTags returns every tag in the system.
func (s *SQLiteStore) ListAllTags(ctx context.Context) ([]core.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list all tags: %w", err)
	}
	return collectTags(rows)
}

// ListFeedTags returns every feed-tag association grouped by feed ID.
func (s *SQLiteStore) ListFeedTags(ctx context.Context) (map[int64][]core.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ft.feed_id, t.id, t.name
		FROM feed_tags ft
		JOIN tags t ON t.id = ft.tag_id
		ORDER BY ft.feed_id, t.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list feed tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	feedTags := make(map[int64][]core.Tag)
	for rows.Next() {
		var feedID int64
		var tag core.Tag
		if err := rows.Scan(&feedID, &tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan feed tag: %w", err)
		}
		feedTags[feedID] = append(feedTags[feedID], tag)
	}
	return feedTags, rows.Err()
}

// ListFeedsByTag returns all feeds associated with the given tag name.
func (s *SQLiteStore) ListFeedsByTag(ctx context.Context, tagName string) ([]*core.Feed, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE id IN (SELECT ft.feed_id FROM feed_tags ft JOIN tags t ON t.id = ft.tag_id WHERE t.name = ?) ORDER BY id`,
		tagName,
	)
	if err != nil {
		return nil, fmt.Errorf("list feeds by tag: %w", err)
	}
	return collectFeeds(rows)
}
