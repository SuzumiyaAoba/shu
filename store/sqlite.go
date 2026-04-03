package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	sqlite "modernc.org/sqlite"
)

const sqliteConstraintUnique = 2067

// SQLiteOptions controls runtime settings for [SQLiteStore].
type SQLiteOptions struct {
	MaxOpenConns int
	BusyTimeout  time.Duration
}

// migrationsFS embeds all SQL migration files from the migrations directory.
// Files are named with a numeric prefix (e.g. 001_, 002_) and executed in
// lexicographic order. Each migration runs at most once, tracked by the
// schema_migrations table.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// feedColumns is the SELECT column list shared by GetFeed and ListFeeds.
const feedColumns = `id, url, title, site_url, added_at, fetched_at, description, language, image_url, feed_type, etag, last_modified, error_count, last_error, disabled, fetch_interval_sec`

// entryColumns is the SELECT column list shared by ListEntries.
const entryColumns = `id, feed_id, guid, title, link, summary, published_at, fetched_at, content, author, image_url, categories, updated_at, enclosures, authors, links, contributors, rights, source, read_at, starred_at`

const insertEntrySQL = `INSERT OR IGNORE INTO entries (feed_id, guid, title, link, summary, published_at, content, author, image_url, categories, updated_at, enclosures, authors, links, contributors, rights, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type entryFilterQuery struct {
	conditions []string
	args       []any
}

// nowRFC3339 returns the current UTC time formatted as RFC 3339.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// parseNullableTime parses a nullable RFC 3339 timestamp string into *time.Time.
func parseNullableTime(s *string, field string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	return &t, nil
}

// formatNullableTime formats a *time.Time as an RFC 3339 string pointer.
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// SQLiteStore implements [Store] using a SQLite database via the pure-Go
// modernc.org/sqlite driver (no CGo required).
//
// On initialization it enables WAL journal mode for better read concurrency,
// turns on foreign key enforcement (required for ON DELETE CASCADE), and
// applies the embedded schema migrations. The underlying [sql.DB] is safe for
// concurrent use; the store keeps max open connections at 1 to avoid SQLite
// lock contention while still allowing concurrent callers through the pool.
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
	return NewSQLiteStoreWithOptions(dsn, nil)
}

// NewSQLiteStoreWithOptions opens (or creates) a SQLite database using the
// provided options.
func NewSQLiteStoreWithOptions(dsn string, options *SQLiteOptions) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	maxOpenConns := 1
	busyTimeout := 5 * time.Second
	if options != nil {
		if options.MaxOpenConns > 0 {
			maxOpenConns = options.MaxOpenConns
		}
		if options.BusyTimeout > 0 {
			busyTimeout = options.BusyTimeout
		}
	}

	// SQLite supports limited concurrency. Use a single connection to avoid
	// "database is locked" errors, especially with in-memory databases.
	db.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout/time.Millisecond)); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
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
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
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
		feed.URL, feed.Title, feed.SiteURL, nowRFC3339(),
		feed.Description, feed.Language, feed.ImageURL, feed.FeedType,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique {
			return fmt.Errorf("feed %s: %w", feed.URL, core.ErrFeedAlreadyExists)
		}
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
	return fetchFeed(row, fmt.Sprintf("feed %d", id))
}

// GetFeedByURL retrieves a single feed by its unique URL.
func (s *SQLiteStore) GetFeedByURL(ctx context.Context, url string) (*core.Feed, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE url = ?`, url,
	)
	return fetchFeed(row, fmt.Sprintf("feed %s", url))
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
	return collectFeeds(rows)
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
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET fetched_at = ? WHERE id = ?`, nowRFC3339(), id,
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

	inserted, err := addEntriesTx(ctx, tx, entries)
	if err != nil {
		return 0, err
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
// GetEntry retrieves a single entry by its primary key.
func (s *SQLiteStore) GetEntry(ctx context.Context, id int64) (*core.Entry, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+entryColumns+` FROM entries WHERE id = ?`, id)
	return fetchEntry(row, fmt.Sprintf("entry %d", id))
}

// Results are always ordered by fetched_at DESC (newest first).
func (s *SQLiteStore) ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error) {
	query, args := newEntryFilterQuery(filter).buildSelectEntries(filter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	return collectEntries(rows)
}

// CountEntries returns the total number of entries matching the filter.
func (s *SQLiteStore) CountEntries(ctx context.Context, filter core.EntryFilter) (int, error) {
	query, args := newEntryFilterQuery(filter).buildCountEntries()

	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count entries: %w", err)
	}
	return count, nil
}

func buildEntryConditions(filter core.EntryFilter) ([]string, []any) {
	query := newEntryFilterQuery(filter)
	return query.conditions, query.args
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

	var disabled int
	if err := s.Scan(
		&f.ID, &f.URL, &f.Title, &f.SiteURL, &addedAt, &fetchedAt,
		&f.Description, &f.Language, &f.ImageURL, &f.FeedType,
		&f.ETag, &f.LastModified,
		&f.ErrorCount, &f.LastError, &disabled, &f.FetchIntervalSec,
	); err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}

	t, err := time.Parse(time.RFC3339, addedAt)
	if err != nil {
		return nil, fmt.Errorf("parse added_at: %w", err)
	}
	f.AddedAt = t

	if f.FetchedAt, err = parseNullableTime(fetchedAt, "fetched_at"); err != nil {
		return nil, err
	}

	f.Disabled = disabled != 0

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
	var readAt *string
	var starredAt *string

	var categories, enclosures, authors, links, contributors, source string

	if err := s.Scan(
		&e.ID, &e.FeedID, &e.GUID, &e.Title, &e.Link, &e.Summary,
		&publishedAt, &fetchedAt,
		&e.Content, &e.Author, &e.ImageURL, &categories, &updatedAt, &enclosures,
		&authors, &links, &contributors, &e.Rights, &source,
		&readAt, &starredAt,
	); err != nil {
		return nil, fmt.Errorf("scan entry: %w", err)
	}

	t, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return nil, fmt.Errorf("parse fetched_at: %w", err)
	}
	e.FetchedAt = t

	if e.PublishedAt, err = parseNullableTime(publishedAt, "published_at"); err != nil {
		return nil, err
	}
	if e.UpdatedAt, err = parseNullableTime(updatedAt, "updated_at"); err != nil {
		return nil, err
	}
	if e.ReadAt, err = parseNullableTime(readAt, "read_at"); err != nil {
		return nil, err
	}
	if e.StarredAt, err = parseNullableTime(starredAt, "starred_at"); err != nil {
		return nil, err
	}

	e.Categories = []byte(categories)
	if len(e.Categories) == 0 {
		e.Categories = nil
	}
	e.Enclosures = []byte(enclosures)
	if len(e.Enclosures) == 0 {
		e.Enclosures = nil
	}
	e.Authors = []byte(authors)
	if len(e.Authors) == 0 {
		e.Authors = nil
	}
	e.Links = []byte(links)
	if len(e.Links) == 0 {
		e.Links = nil
	}
	e.Contributors = []byte(contributors)
	if len(e.Contributors) == 0 {
		e.Contributors = nil
	}
	e.Source = []byte(source)
	if len(e.Source) == 0 {
		e.Source = nil
	}

	return &e, nil
}

func fetchFeed(s scanner, label string) (*core.Feed, error) {
	feed, err := scanFeed(s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", label, core.ErrFeedNotFound)
		}
		return nil, err
	}
	return feed, nil
}

func fetchEntry(s scanner, label string) (*core.Entry, error) {
	entry, err := scanEntry(s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", label, core.ErrEntryNotFound)
		}
		return nil, err
	}
	return entry, nil
}

func collectFeeds(rows *sql.Rows) ([]*core.Feed, error) {
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

func collectEntries(rows *sql.Rows) ([]*core.Entry, error) {
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

func collectTags(rows *sql.Rows) ([]core.Tag, error) {
	defer rows.Close()

	var tags []core.Tag
	for rows.Next() {
		var t core.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func newEntryFilterQuery(filter core.EntryFilter) entryFilterQuery {
	query := entryFilterQuery{}

	if filter.FeedID != nil {
		query.add(`feed_id = ?`, *filter.FeedID)
	}
	if filter.UnreadOnly {
		query.add(`read_at IS NULL`)
	}
	if filter.Tag != "" {
		query.add(`feed_id IN (SELECT ft.feed_id FROM feed_tags ft JOIN tags t ON t.id = ft.tag_id WHERE t.name = ?)`, filter.Tag)
	}
	if filter.StarredOnly {
		query.add(`starred_at IS NOT NULL`)
	}

	return query
}

func (q *entryFilterQuery) add(condition string, args ...any) {
	q.conditions = append(q.conditions, condition)
	q.args = append(q.args, args...)
}

func (q entryFilterQuery) buildSelectEntries(filter core.EntryFilter) (string, []any) {
	query := `SELECT ` + entryColumns + ` FROM entries` + q.whereClause() + ` ORDER BY fetched_at DESC, id DESC`
	args := q.cloneArgs()

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	return query, args
}

func (q entryFilterQuery) buildCountEntries() (string, []any) {
	return `SELECT COUNT(*) FROM entries` + q.whereClause(), q.cloneArgs()
}

func (q entryFilterQuery) whereClause() string {
	if len(q.conditions) == 0 {
		return ""
	}
	return ` WHERE ` + strings.Join(q.conditions, ` AND `)
}

func (q entryFilterQuery) cloneArgs() []any {
	return append([]any(nil), q.args...)
}

// UpdateFeed updates mutable feed fields. Only non-nil fields in the update
// struct are applied.
func (s *SQLiteStore) UpdateFeed(ctx context.Context, id int64, update core.FeedUpdate) error {
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

	if len(sets) == 0 {
		return nil
	}

	args = append(args, id)
	query := "UPDATE feeds SET " + strings.Join(sets, ", ") + " WHERE id = ?"

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	return nil
}

// UpdateFeedCacheHeaders stores the HTTP ETag and Last-Modified headers
// received during a fetch.
func (s *SQLiteStore) UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET etag = ?, last_modified = ? WHERE id = ?`,
		etag, lastModified, id,
	)
	if err != nil {
		return fmt.Errorf("update cache headers: %w", err)
	}
	return nil
}

func buildIDPlaceholders(ids []int64) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	return strings.Join(placeholders, ", "), args
}

func buildEntriesColumnUpdate(column string, value any, ids []int64) (string, []any) {
	placeholders, args := buildIDPlaceholders(ids)
	if value == nil {
		return fmt.Sprintf(`UPDATE entries SET %s = NULL WHERE id IN (%s)`, column, placeholders), args
	}
	return fmt.Sprintf(`UPDATE entries SET %s = ? WHERE id IN (%s)`, column, placeholders), append([]any{value}, args...)
}

func (s *SQLiteStore) updateEntriesColumn(ctx context.Context, column string, value any, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query, args := buildEntriesColumnUpdate(column, value, ids)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLiteStore) updateEntryState(ctx context.Context, column string, value any, ids []int64, label string) error {
	if err := s.updateEntriesColumn(ctx, column, value, ids); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

// MarkEntryRead sets the read_at timestamp on the given entry.
func (s *SQLiteStore) MarkEntryRead(ctx context.Context, id int64) error {
	return s.updateEntryState(ctx, "read_at", nowRFC3339(), []int64{id}, "mark entry read")
}

// MarkEntriesRead sets the read_at timestamp on the given entries.
func (s *SQLiteStore) MarkEntriesRead(ctx context.Context, ids []int64) error {
	return s.updateEntryState(ctx, "read_at", nowRFC3339(), ids, "mark entries read")
}

// MarkEntryUnread clears the read_at timestamp on the given entry.
func (s *SQLiteStore) MarkEntryUnread(ctx context.Context, id int64) error {
	return s.updateEntryState(ctx, "read_at", nil, []int64{id}, "mark entry unread")
}

// MarkEntriesUnread clears the read_at timestamp on the given entries.
func (s *SQLiteStore) MarkEntriesUnread(ctx context.Context, ids []int64) error {
	return s.updateEntryState(ctx, "read_at", nil, ids, "mark entries unread")
}

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
	defer rows.Close()

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

// SearchEntries performs full-text search using the FTS5 index.
func (s *SQLiteStore) SearchEntries(ctx context.Context, query string, limit int) ([]*core.Entry, error) {
	return s.SearchEntriesPage(ctx, query, limit, 0)
}

// SearchEntriesPage performs paginated full-text search using the FTS5 index.
func (s *SQLiteStore) SearchEntriesPage(ctx context.Context, query string, limit, offset int) ([]*core.Entry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+entryColumns+` FROM entries WHERE id IN (SELECT rowid FROM entries_fts WHERE entries_fts MATCH ?) ORDER BY fetched_at DESC LIMIT ? OFFSET ?`,
		query, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search entries: %w", err)
	}
	return collectEntries(rows)
}

// CountSearchEntries returns the total number of entries matching the FTS query.
func (s *SQLiteStore) CountSearchEntries(ctx context.Context, query string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE id IN (SELECT rowid FROM entries_fts WHERE entries_fts MATCH ?)`,
		query,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count search entries: %w", err)
	}
	return count, nil
}

// StarEntry sets the starred_at timestamp on the given entry.
func (s *SQLiteStore) StarEntry(ctx context.Context, id int64) error {
	return s.updateEntryState(ctx, "starred_at", nowRFC3339(), []int64{id}, "star entry")
}

// StarEntries sets the starred_at timestamp on the given entries.
func (s *SQLiteStore) StarEntries(ctx context.Context, ids []int64) error {
	return s.updateEntryState(ctx, "starred_at", nowRFC3339(), ids, "star entries")
}

// UnstarEntry clears the starred_at timestamp on the given entry.
func (s *SQLiteStore) UnstarEntry(ctx context.Context, id int64) error {
	return s.updateEntryState(ctx, "starred_at", nil, []int64{id}, "unstar entry")
}

// UnstarEntries clears the starred_at timestamp on the given entries.
func (s *SQLiteStore) UnstarEntries(ctx context.Context, ids []int64) error {
	return s.updateEntryState(ctx, "starred_at", nil, ids, "unstar entries")
}

// maxErrorCount is the number of consecutive failures before a feed is
// automatically disabled.
const maxErrorCount = 5

// RecordFeedError increments the error count and stores the error message.
// If the count reaches maxErrorCount the feed is automatically disabled.
func (s *SQLiteStore) RecordFeedError(ctx context.Context, id int64, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
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
	_, err := s.db.ExecContext(ctx,
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
	_, err := s.db.ExecContext(ctx, `UPDATE feeds SET disabled = ? WHERE id = ?`, val, id)
	if err != nil {
		return fmt.Errorf("set feed disabled: %w", err)
	}
	return nil
}

// FeedStats returns aggregate statistics for all feeds.
func (s *SQLiteStore) FeedStats(ctx context.Context) ([]core.FeedStats, error) {
	rows, err := s.db.QueryContext(ctx, `
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
	defer rows.Close()

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
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE fetched_at < ? AND starred_at IS NULL`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup entries: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// FindDuplicateEntries returns entries from other feeds that share the same
// link URL as the given entry.
func (s *SQLiteStore) FindDuplicateEntries(ctx context.Context, entryID int64) ([]*core.Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+entryColumns+` FROM entries WHERE link = (SELECT link FROM entries WHERE id = ?) AND id != ? AND link != ''`,
		entryID, entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("find duplicates: %w", err)
	}
	return collectEntries(rows)
}

func addEntriesTx(ctx context.Context, tx *sql.Tx, entries []*core.Entry) (int, error) {
	stmt, err := tx.PrepareContext(ctx, insertEntrySQL)
	if err != nil {
		return 0, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, entry := range entries {
		count, err := insertEntry(ctx, stmt, entry)
		if err != nil {
			return 0, err
		}
		inserted += count
	}

	return inserted, nil
}

func insertEntry(ctx context.Context, stmt *sql.Stmt, entry *core.Entry) (int, error) {
	result, err := stmt.ExecContext(ctx,
		entry.FeedID, entry.GUID, entry.Title, entry.Link, entry.Summary, formatNullableTime(entry.PublishedAt),
		entry.Content, entry.Author, entry.ImageURL, string(entry.Categories), formatNullableTime(entry.UpdatedAt), string(entry.Enclosures),
		string(entry.Authors), string(entry.Links), string(entry.Contributors), entry.Rights, string(entry.Source),
	)
	if err != nil {
		return 0, fmt.Errorf("insert entry: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(rowsAffected), nil
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
