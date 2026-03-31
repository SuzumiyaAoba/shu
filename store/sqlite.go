package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	_ "modernc.org/sqlite" // Register the pure-Go SQLite driver.
)

// migrationsFS embeds all SQL migration files from the migrations directory.
// Files are named with a numeric prefix (e.g. 001_, 002_) and executed in
// lexicographic order. Each migration runs at most once, tracked by the
// schema_migrations table.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// feedColumns is the SELECT column list shared by GetFeed and ListFeeds.
const feedColumns = `id, url, title, site_url, added_at, fetched_at, description, language, image_url, feed_type`

// entryColumns is the SELECT column list shared by ListEntries.
const entryColumns = `id, feed_id, guid, title, link, summary, published_at, fetched_at, content, author, image_url, categories, updated_at, enclosures`

// SQLiteStore implements [Store] using a SQLite database via the pure-Go
// modernc.org/sqlite driver (no CGo required).
//
// On initialization it enables WAL journal mode for better read concurrency,
// turns on foreign key enforcement (required for ON DELETE CASCADE), and
// applies the embedded schema migrations.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at the given DSN and
// returns a ready-to-use [SQLiteStore].
//
// The DSN is typically a file path (e.g. "/home/user/.shu/shu.db") or the
// special value ":memory:" for an in-memory database used in tests.
//
// Initialization steps:
//  1. Open the database connection.
//  2. Enable WAL journal mode for improved concurrent read performance.
//  3. Enable foreign key constraint enforcement (SQLite disables it by default).
//  4. Run all pending schema migrations tracked by the schema_migrations table.
//
// If any step fails, the database is closed and an error is returned.
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

	s := &SQLiteStore{db: db}
	if err := s.runMigrations(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// runMigrations applies all embedded SQL migration files that have not yet been
// executed. It uses a schema_migrations table to track which files have already
// been applied, ensuring each migration runs exactly once and enabling safe
// ALTER TABLE statements that would fail if re-executed.
func (s *SQLiteStore) runMigrations() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		name := entry.Name()

		var count int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, name).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		if _, err := s.db.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}

// Close closes the underlying database connection pool.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// AddFeed inserts a new feed row. On success the feed's ID is set to the
// auto-generated primary key and AddedAt is set to the current UTC time.
// Returns an error if a feed with the same URL already exists (UNIQUE
// constraint violation).
func (s *SQLiteStore) AddFeed(ctx context.Context, feed *core.Feed) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO feeds (url, title, site_url, added_at, description, language, image_url, feed_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		feed.URL, feed.Title, feed.SiteURL, now.Format(time.RFC3339),
		feed.Description, feed.Language, feed.ImageURL, feed.FeedType,
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

// GetFeed retrieves a single feed by its primary key.
// Returns a "scan feed" error wrapping sql.ErrNoRows if the ID does not exist.
func (s *SQLiteStore) GetFeed(ctx context.Context, id int64) (*core.Feed, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE id = ?`, id,
	)
	return scanFeed(row)
}

// ListFeeds returns all registered feeds ordered by ascending ID.
// Returns an empty (non-nil) slice if no feeds are registered.
func (s *SQLiteStore) ListFeeds(ctx context.Context) ([]*core.Feed, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+feedColumns+` FROM feeds ORDER BY id`,
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

// RemoveFeed deletes the feed with the given ID. Due to the ON DELETE CASCADE
// constraint on the entries table, all entries belonging to this feed are also
// deleted. No error is returned if the ID does not exist.
func (s *SQLiteStore) RemoveFeed(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	return nil
}

// UpdateFeedFetchedAt sets the feed's fetched_at column to the current UTC
// time. This is called after a successful fetch cycle to record when the feed
// was last refreshed.
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

// AddEntries inserts multiple entries in a single transaction. Entries whose
// (feed_id, guid) pair already exists in the database are silently skipped
// thanks to INSERT OR IGNORE and the UNIQUE constraint.
//
// Returns the number of rows actually inserted (i.e. excluding duplicates).
// If the input slice is empty or nil, it returns (0, nil) immediately without
// opening a transaction.
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
		`INSERT OR IGNORE INTO entries (feed_id, guid, title, link, summary, published_at, content, author, image_url, categories, updated_at, enclosures) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		var updAt *string
		if e.UpdatedAt != nil {
			s := e.UpdatedAt.UTC().Format(time.RFC3339)
			updAt = &s
		}
		result, err := stmt.ExecContext(ctx,
			e.FeedID, e.GUID, e.Title, e.Link, e.Summary, pubAt,
			e.Content, e.Author, e.ImageURL, e.Categories, updAt, e.Enclosures,
		)
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

// ListEntries queries entries matching the given filter criteria.
//
// The filter supports:
//   - FeedID: when non-nil, restricts results to the specified feed.
//   - Limit: caps the number of rows returned (0 = unlimited).
//   - Offset: skips the first N rows for pagination.
//
// Results are always ordered by fetched_at DESC (newest first).
func (s *SQLiteStore) ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error) {
	query := `SELECT ` + entryColumns + ` FROM entries`
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

// scanner is a minimal interface satisfied by both *sql.Row and *sql.Rows,
// allowing scanFeed and scanEntry to work with either single-row or multi-row
// query results.
type scanner interface {
	Scan(dest ...any) error
}

// scanFeed reads a single feed row from the scanner and converts the stored
// ISO 8601 timestamp strings back into Go time.Time values.
func scanFeed(s scanner) (*core.Feed, error) {
	var f core.Feed
	var addedAt string
	var fetchedAt *string

	if err := s.Scan(
		&f.ID, &f.URL, &f.Title, &f.SiteURL, &addedAt, &fetchedAt,
		&f.Description, &f.Language, &f.ImageURL, &f.FeedType,
	); err != nil {
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

// scanEntry reads a single entry row from the scanner and converts the stored
// ISO 8601 timestamp strings back into Go time.Time values. The published_at
// and updated_at columns are nullable and map to *time.Time.
func scanEntry(s scanner) (*core.Entry, error) {
	var e core.Entry
	var publishedAt *string
	var fetchedAt string
	var updatedAt *string

	if err := s.Scan(
		&e.ID, &e.FeedID, &e.GUID, &e.Title, &e.Link, &e.Summary,
		&publishedAt, &fetchedAt,
		&e.Content, &e.Author, &e.ImageURL, &e.Categories, &updatedAt, &e.Enclosures,
	); err != nil {
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

	if updatedAt != nil {
		t, err := time.Parse(time.RFC3339, *updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at: %w", err)
		}
		e.UpdatedAt = &t
	}

	return &e, nil
}
