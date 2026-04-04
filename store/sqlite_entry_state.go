package store

import (
	"context"
	"fmt"
	"strings"
)

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
	_, err := s.executor(ctx).ExecContext(ctx, query, args...)
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
