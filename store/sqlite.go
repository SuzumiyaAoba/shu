package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSQL string

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(initSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) AddFeed(ctx context.Context, feed *core.Feed) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO feeds (url, title, site_url, added_at) VALUES (?, ?, ?, ?)`,
		feed.URL, feed.Title, feed.SiteURL, now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert feed: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	feed.ID = id
	feed.AddedAt = now
	return nil
}

func (s *SQLiteStore) GetFeed(ctx context.Context, id int64) (*core.Feed, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, url, title, site_url, added_at, fetched_at FROM feeds WHERE id = ?`, id,
	)
	return scanFeed(row)
}

func (s *SQLiteStore) ListFeeds(ctx context.Context) ([]*core.Feed, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, url, title, site_url, added_at, fetched_at FROM feeds ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*core.Feed
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

func (s *SQLiteStore) RemoveFeed(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpdateFeedFetchedAt(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET fetched_at = ? WHERE id = ?`, now, id,
	)
	if err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

func (s *SQLiteStore) AddEntries(ctx context.Context, entries []*core.Entry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO entries (feed_id, guid, title, link, summary, published_at) VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, e := range entries {
		var pubAt *string
		if e.PublishedAt != nil {
			s := e.PublishedAt.UTC().Format(time.RFC3339)
			pubAt = &s
		}
		result, err := stmt.ExecContext(ctx, e.FeedID, e.GUID, e.Title, e.Link, e.Summary, pubAt)
		if err != nil {
			return 0, fmt.Errorf("insert entry: %w", err)
		}
		ra, _ := result.RowsAffected()
		inserted += int(ra)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

func (s *SQLiteStore) ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error) {
	query := `SELECT id, feed_id, guid, title, link, summary, published_at, fetched_at FROM entries`
	var args []any

	if filter.FeedID != nil {
		query += ` WHERE feed_id = ?`
		args = append(args, *filter.FeedID)
	}

	query += ` ORDER BY fetched_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []*core.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFeed(s scanner) (*core.Feed, error) {
	var f core.Feed
	var addedAt string
	var fetchedAt *string

	if err := s.Scan(&f.ID, &f.URL, &f.Title, &f.SiteURL, &addedAt, &fetchedAt); err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}

	t, err := time.Parse(time.RFC3339, addedAt)
	if err != nil {
		return nil, fmt.Errorf("parse added_at: %w", err)
	}
	f.AddedAt = t

	if fetchedAt != nil {
		t, err := time.Parse(time.RFC3339, *fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse fetched_at: %w", err)
		}
		f.FetchedAt = &t
	}

	return &f, nil
}

func scanEntry(s scanner) (*core.Entry, error) {
	var e core.Entry
	var publishedAt *string
	var fetchedAt string

	if err := s.Scan(&e.ID, &e.FeedID, &e.GUID, &e.Title, &e.Link, &e.Summary, &publishedAt, &fetchedAt); err != nil {
		return nil, fmt.Errorf("scan entry: %w", err)
	}

	t, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return nil, fmt.Errorf("parse fetched_at: %w", err)
	}
	e.FetchedAt = t

	if publishedAt != nil {
		t, err := time.Parse(time.RFC3339, *publishedAt)
		if err != nil {
			return nil, fmt.Errorf("parse published_at: %w", err)
		}
		e.PublishedAt = &t
	}

	return &e, nil
}
