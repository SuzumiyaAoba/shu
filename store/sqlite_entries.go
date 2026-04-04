package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/SuzumiyaAoba/shu/core"
)

type entryFilterQuery struct {
	conditions []string
	args       []any
}

// AddEntries inserts multiple entries in a single transaction. Entries whose
// (feed_id, guid) pair already exists in the database are silently skipped
// thanks to INSERT OR IGNORE and the UNIQUE constraint.
//
// Returns the number of rows actually inserted (i.e. excluding duplicates).
// If the input slice is empty or nil, it returns (0, nil) immediately without
// opening a transaction. Participates in an outer transaction if one is
// present in ctx.
func (s *SQLiteStore) AddEntries(ctx context.Context, entries []*core.Entry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	inserted := 0
	err := s.RunInTx(ctx, func(ctx context.Context) error {
		var err error
		inserted, err = addEntriesEx(ctx, s.executor(ctx), entries)
		return err
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

// GetEntry retrieves a single entry by its primary key.
func (s *SQLiteStore) GetEntry(ctx context.Context, id int64) (*core.Entry, error) {
	row := s.executor(ctx).QueryRowContext(ctx, `SELECT `+entryColumns+` FROM entries WHERE id = ?`, id)
	return fetchEntry(row, fmt.Sprintf("entry %d", id))
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
	query, args := newEntryFilterQuery(filter).buildSelectEntries(filter)

	rows, err := s.executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	return collectEntries(rows)
}

// CountEntries returns the total number of entries matching the filter.
func (s *SQLiteStore) CountEntries(ctx context.Context, filter core.EntryFilter) (int, error) {
	query, args := newEntryFilterQuery(filter).buildCountEntries()

	var count int
	if err := s.executor(ctx).QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count entries: %w", err)
	}
	return count, nil
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

// SearchEntries performs full-text search using the FTS5 index.
func (s *SQLiteStore) SearchEntries(ctx context.Context, query string, limit int) ([]*core.Entry, error) {
	return s.SearchEntriesPage(ctx, query, limit, 0)
}

// SearchEntriesPage performs paginated full-text search using the FTS5 index.
func (s *SQLiteStore) SearchEntriesPage(ctx context.Context, query string, limit, offset int) ([]*core.Entry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.executor(ctx).QueryContext(ctx,
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
	if err := s.executor(ctx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE id IN (SELECT rowid FROM entries_fts WHERE entries_fts MATCH ?)`,
		query,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count search entries: %w", err)
	}
	return count, nil
}

// FindDuplicateEntries returns entries from other feeds that share the same
// link URL as the given entry.
func (s *SQLiteStore) FindDuplicateEntries(ctx context.Context, entryID int64) ([]*core.Entry, error) {
	rows, err := s.executor(ctx).QueryContext(ctx,
		`SELECT `+entryColumns+` FROM entries WHERE link = (SELECT link FROM entries WHERE id = ?) AND id != ? AND link != ''`,
		entryID, entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("find duplicates: %w", err)
	}
	return collectEntries(rows)
}

func addEntriesEx(ctx context.Context, ex sqlExecutor, entries []*core.Entry) (int, error) {
	stmt, err := ex.PrepareContext(ctx, insertEntrySQL)
	if err != nil {
		return 0, fmt.Errorf("prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

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
