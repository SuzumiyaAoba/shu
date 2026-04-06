# 013: Database Performance Optimization

## Overview

Optimize SQLite settings and indexes that directly impact performance.
Enable WAL mode, add missing indexes, and configure the default busy timeout.

---

## Proposal 1: Enable SQLite WAL Mode

### Current Issue

SQLite's default journal mode is `delete`, which causes write and read operations to block each other.
Since `shu fetch` adds entries concurrently with multiple workers, `SQLITE_BUSY` errors occur frequently.

### Proposed Change

Enable WAL mode after establishing the database connection:

```go
// store/sqlite.go
func (s *SQLiteStore) configurePragmas() error {
    pragmas := []string{
        `PRAGMA journal_mode = WAL`,
        `PRAGMA synchronous = NORMAL`,
        `PRAGMA foreign_keys = ON`,
    }
    for _, p := range pragmas {
        if _, err := s.db.Exec(p); err != nil {
            return fmt.Errorf("exec %s: %w", p, err)
        }
    }
    return nil
}
```

### Notes

- WAL mode, once set, persists at the database file level
- `synchronous = NORMAL` is safe in WAL mode and faster than `FULL`
- `foreign_keys = ON` should be verified as already set in existing code

### Impact Scope

| File | Change |
|------|--------|
| `store/sqlite.go` | Add PRAGMA configuration function, call from `NewSQLiteStore` |
| `store/sqlite_test.go` | Add test to verify WAL mode is enabled |

### Effort: Small (1 hour)

---

## Proposal 2: Add Indexes for Entry State Filtering

### Current Issue

The `entries` table has indexes on `feed_id` and `published_at`, but common filter conditions lack indexes:

- `read_at IS NULL` (`--unread` filter): No index
- `starred_at IS NOT NULL` (`--starred` filter): No index

As the number of entries grows, these filter queries degrade in performance.

### Proposed Change

Add a new migration file:

```sql
-- +goose Up
-- Partial indexes for common entry state filters.
CREATE INDEX IF NOT EXISTS idx_entries_unread ON entries(fetched_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_entries_starred ON entries(starred_at DESC)
    WHERE starred_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_entries_starred;
DROP INDEX IF EXISTS idx_entries_unread;
```

### Notes

- SQLite's partial indexes significantly speed up conditional queries
- Index size is proportional to the number of unread/starred entries, smaller than a full index
- For existing data, `CREATE INDEX IF NOT EXISTS` automatically builds the index during migration

### Impact Scope

| File | Change |
|------|--------|
| `store/migrations/009_entry_state_indexes.sql` | New: Partial index definitions |

### Effort: Tiny (15 minutes)

---

## Proposal 3: Set Default Busy Timeout

### Current Issue

The default value of `--sqlite-busy-timeout` is `0` (no timeout) (`cmd/root.go:90`).
This means `shu fetch` workers immediately return `SQLITE_BUSY` errors on database lock contention.

This issue occurs unless users explicitly specify `--sqlite-busy-timeout 5s`.

### Proposed Change

Change the default value to `5s`:

```go
rootCmd.PersistentFlags().DurationVar(&sqliteBusyTimeout, "sqlite-busy-timeout", 5*time.Second, "SQLite busy timeout (e.g. 5s)")
```

Also adjust the zero value check in `sqliteOptionsFromFlags` to ensure the default is properly applied.

### Impact Scope

| File | Change |
|------|--------|
| `cmd/root.go` | Change `--sqlite-busy-timeout` default to `5s` |
| `cmd/root.go` | Adjust zero value check in `sqliteOptionsFromFlags` |

### Effort: Tiny (15 minutes)

---

## Proposal 4: Optimize FTS5 Search Query

### Current Issue

The FTS5 search at `store/sqlite_entries.go:166-168` uses a subquery + JOIN pattern:

```sql
SELECT ... FROM entries WHERE id IN (SELECT rowid FROM entries_fts WHERE entries_fts MATCH ?)
ORDER BY fetched_at DESC LIMIT ? OFFSET ?
```

In this pattern, the entire FTS5 result is generated in the subquery before sorting occurs.
Performance may degrade when entry count is large.

### Proposed Change

Verify alternative queries using FTS5's ranking features, and apply only if performance improves:

```sql
SELECT ... FROM entries e
INNER JOIN entries_fts f ON e.id = f.rowid
WHERE f.entries_fts MATCH ?
ORDER BY e.fetched_at DESC
LIMIT ? OFFSET ?
```

### Notes

- Actual performance differences must be verified with `EXPLAIN QUERY PLAN`
- Score-based sorting using FTS5's `rank` column is also worth considering
- Avoid making changes without benchmarks

### Impact Scope

| File | Change |
|------|--------|
| `store/sqlite_entries.go` | Change `SearchEntriesPage` query (subject to benchmark results) |
| `store/sqlite_entries_test.go` | Add search performance tests |

### Effort: Medium (2–3 hours, including benchmarks)

---

## Priority Matrix

| Proposal | Impact | Effort | Recommended Priority |
|----------|--------|--------|----------------------|
| 3. busy timeout default | High (prevent SQLITE_BUSY) | Tiny | **High** — Immediate effect |
| 1. WAL mode | High (read-write concurrency) | Small | **High** |
| 2. State filter index | Medium (query optimization) | Tiny | **High** |
| 4. FTS5 query optimization | Medium | Medium | Low — Benchmarks required |

## Recommended Execution Order

### Phase 1: High-impact changes (30 minutes)
1. Proposal 3 — busy timeout default
2. Proposal 2 — State filter index

### Phase 2: Structural improvements (1 hour)
3. Proposal 1 — WAL mode

### Phase 3: Requires verification (2–3 hours)
4. Proposal 4 — FTS5 optimization (apply pending benchmark results)

## Completion Checklist

- [ ] All existing tests pass
- [ ] Test verifying WAL mode is enabled
- [ ] `EXPLAIN QUERY PLAN` confirms indexes are used
- [ ] `golangci-lint run` clean
