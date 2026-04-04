# 001: Introduce Transaction Boundaries for Composite Operations

## Overview

Multiple store calls composed into a single operation can leave the database in an inconsistent state
if one call fails partway through. Introduce transaction boundaries with `RunInTx` to guarantee atomicity
across composite operations.

## Current Problem

### EnableFeed — Two-stage update can partially fail

```go
// core/update.go
func (s *Service) EnableFeed(ctx context.Context, id int64) error {
    s.store.SetFeedDisabled(ctx, id, false) // ← succeeds
    s.store.ResetFeedError(ctx, id)         // ← fails → disabled=false but error_count remains
}
```

### RemoveDeadFeeds — Loop of individual deletions can partially fail

```go
// core/dead.go
func (s *Service) RemoveDeadFeeds(ctx context.Context) ([]*Feed, error) {
    for _, feed := range dead {
        s.RemoveFeed(ctx, feed.ID) // ← fails on 3rd iteration → only 2 feeds deleted
    }
}
```

### persistFetchedFeed — Post-AddEntries operations can fail

```go
// core/fetch_persist.go
func (s *Service) persistFetchedFeed(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error) {
    s.storeConditionalHeaders(ctx, feed.ID, document.headers)
    entries, _ := parseFetchedEntries(...)
    s.store.AddEntries(ctx, entries)    // ← succeeds
    s.store.UpdateFeedFetchedAt(...)    // ← fails → entries saved but fetched_at not updated
    s.store.ResetFeedError(...)
}
```

## Improvement Plan

### Step 1: Add Transaction Support to Store Interface

```go
// core/core.go — add
type TxRunner interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

Compose `TxRunner` into the `Store` interface.

### Step 2: Implement RunInTx in SQLiteStore

```go
// store/sqlite.go
func (s *SQLiteStore) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    // Embed transaction in context, methods detect and use it
    txCtx := contextWithTx(ctx, tx)
    if err := fn(txCtx); err != nil {
        return err
    }
    return tx.Commit()
}
```

Introduce an `executor` helper that each store method uses to detect and use the transaction:

```go
func (s *SQLiteStore) executor(ctx context.Context) executor {
    if tx := txFromContext(ctx); tx != nil {
        return tx
    }
    return s.db
}
```

### Step 3: Wrap Composite Operations in Transactions

```go
func (s *Service) EnableFeed(ctx context.Context, id int64) error {
    return s.store.RunInTx(ctx, func(ctx context.Context) error {
        if err := s.store.SetFeedDisabled(ctx, id, false); err != nil {
            return fmt.Errorf("enable feed %d: %w", id, err)
        }
        if err := s.store.ResetFeedError(ctx, id); err != nil {
            return fmt.Errorf("reset errors feed %d: %w", id, err)
        }
        return nil
    })
}
```

### Step 4: Integrate Existing AddEntries Transaction Management

Currently `AddEntries` manages its own transaction. Unify it with the `RunInTx` base,
allowing it to participate in an outer transaction (avoiding nested transaction overhead).

## Target Files

| File | Change |
|------|--------|
| `core/core.go` | Add `TxRunner` interface, compose into `Store` |
| `store/sqlite.go` | Implement `RunInTx`, `executor` helper |
| `store/sqlite_feed.go` | Migrate to use `executor(ctx)` |
| `store/sqlite_entries.go` | Change `AddEntries` to be `RunInTx`-aware |
| `store/sqlite_entry_state.go` | Use `executor(ctx)` |
| `store/sqlite_tags.go` | Use `executor(ctx)` |
| `store/sqlite_maintenance.go` | Use `executor(ctx)` |
| `core/update.go` | Wrap `EnableFeed` in transaction |
| `core/dead.go` | Wrap `RemoveDeadFeeds` in transaction |
| `core/fetch_persist.go` | Wrap `persistFetchedFeed` in transaction |
| Test fakes | Add `RunInTx` stub implementations |

## Risks

- Embedding transaction in context is a Go convention but creates implicit dependencies
- Must decide how to handle nested transactions (calling `RunInTx` inside a `RunInTx`)
  - Recommendation: reuse the outer transaction (no SAVEPOINT)
- SQLite's single-writer constraint means long-lived transactions should be avoided

## Completion Criteria

- [x] Test confirms `EnableFeed` rolls back on partial failure
- [x] Test confirms `RemoveDeadFeeds` rolls back on partial failure
- [x] All existing tests pass
- [x] `AddEntries` can participate in outer transaction
