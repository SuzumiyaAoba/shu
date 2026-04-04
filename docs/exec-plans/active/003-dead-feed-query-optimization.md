# 003: Optimize Dead Feed Query

## Overview

`ListDeadFeeds` loads all feeds into memory, then filters in Go.
This can be solved with a SQL-level `WHERE` clause. Add a specialized Store method
to improve efficiency and reduce memory overhead.

## Current Problem

```go
// core/dead.go
func (s *Service) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
    feeds, err := s.store.ListFeeds(ctx)  // Load all N feeds
    if err != nil {
        return nil, err
    }

    dead := make([]*Feed, 0)
    for _, feed := range feeds {
        if feed.ErrorCount > 0 {          // Filter on Go side
            dead = append(dead, feed)
        }
    }
    return dead, nil
}
```

As feed count grows, unnecessary data is loaded into memory.

## Improvement Plan

### Step 1: Add Method to Store Interface

```go
// core/core.go — add to FeedHealthStore
type FeedHealthStore interface {
    RecordFeedError(ctx context.Context, id int64, errMsg string) error
    ResetFeedError(ctx context.Context, id int64) error
    SetFeedDisabled(ctx context.Context, id int64, disabled bool) error
    ListDeadFeeds(ctx context.Context) ([]*Feed, error)  // Add
}
```

### Step 2: Implement in SQLiteStore

```go
// store/sqlite_feed.go
func (s *SQLiteStore) ListDeadFeeds(ctx context.Context) ([]*core.Feed, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT `+feedColumns+` FROM feeds WHERE error_count > 0 ORDER BY id`,
    )
    if err != nil {
        return nil, fmt.Errorf("query dead feeds: %w", err)
    }
    return collectFeeds(rows)
}
```

### Step 3: Simplify core/dead.go

```go
func (s *Service) ListDeadFeeds(ctx context.Context) ([]*Feed, error) {
    return s.store.ListDeadFeeds(ctx)
}
```

### Step 4: Update Test Fakes

Add `ListDeadFeeds` stub to all test fake stores (`fakeStore`, `entryStateErrorStore`).

## Target Files

| File | Change |
|------|--------|
| `core/core.go` | Add `ListDeadFeeds` to `FeedHealthStore` |
| `store/sqlite_feed.go` | Implement SQL query |
| `core/dead.go` | Remove Go-side filtering, delegate to store |
| `app/test_helpers_test.go` | Add stub to `fakeStore` |
| `core/read_test.go` | Add stub to `entryStateErrorStore` |
| `core/dead_test.go` | Update tests for new behavior |

## Risks

- Store interface grows by one method, affecting all fakes
  - Mitigated by plan #004 (Store interface segregation)
- Future logic changes (e.g., different `error_count` threshold) may require duplication

## Completion Criteria

- [ ] `ListDeadFeeds` filters at SQL level
- [ ] No full feed load in `ListDeadFeeds`
- [ ] All existing tests pass
- [ ] Dead feed tests verify SQL filtering
