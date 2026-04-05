# 010: Modern Go Best Practices — Leveraging Go 1.20–1.26 Features

## Overview

This plan identifies opportunities to leverage modern Go standard library features
(Go 1.20+) and best practices in the shu codebase, which already runs Go 1.26.1.
Each proposal is independently implementable and preserves the existing three-layer
architecture (`cmd/` → `core/` → `store/`).

---

## Proposal 1: Adopt `slices` Package (Go 1.21)

### Current Problem

`core/opml.go` uses the legacy `sort.Strings` function and manual slice cloning
via `append([]string{}, parentTags...)`.

### Proposed Change

- `sort.Strings(tagNames)` → `slices.Sort(tagNames)`
- `append([]string{}, parentTags...)` → `slices.Clone(parentTags)`

### Affected Files

| File | Change |
|------|--------|
| `core/opml.go` | Replace `sort` → `slices.Sort`, use `slices.Clone` |

### Effort: Tiny (30 min)

---

## Proposal 2: Aggregate Errors with `errors.Join` (Go 1.20)

### Current Problem

`FetchAll` fetches feeds concurrently but individual feed errors are only logged,
never returned to the caller. This limits observability — the caller cannot
distinguish "all feeds succeeded" from "5 feeds failed silently".

### Proposed Change

Collect per-feed errors during concurrent fetch and return them via `errors.Join`:

```go
// fetch_batch.go
var (
    fetchErrors []error
    mu          sync.Mutex
)
// ... inside worker
if err != nil {
    mu.Lock()
    fetchErrors = append(fetchErrors, err)
    mu.Unlock()
    continue
}
// ... after workers complete
return int(total.Load()), errors.Join(fetchErrors...)
```

### Caveats

- Existing behavior (continue processing remaining feeds on error) is preserved
- The `int` return value (new entry count) remains accurate
- Callers can inspect individual feed errors via `errors.Is(err, &FeedError{})`

### Affected Files

| File | Change |
|------|--------|
| `core/fetch_batch.go` | Collect errors in `fetchFeedsConcurrently`, return via `errors.Join` |
| `cmd/feed_commands.go` | Handle partial-success error (display count + warnings) |
| `core/fetch_batch_test.go` | Add test for partial error return |

### Effort: Small (1–2 hours)

---

## Proposal 3: Generic Row Collection in Store Layer

### Current Problem

`store/sqlite_scan.go` contains three nearly identical functions: `collectFeeds`,
`collectEntries`, and `collectTags`. Each follows the same pattern:
close rows → iterate → scan → append → check rows.Err().

### Proposed Change

Introduce generic `collectRows` and `collectValues` helpers:

```go
func collectRows[T any](rows *sql.Rows, scan func(scanner) (*T, error)) ([]*T, error) {
    defer func() { _ = rows.Close() }()
    var result []*T
    for rows.Next() {
        v, err := scan(rows)
        if err != nil {
            return nil, err
        }
        result = append(result, v)
    }
    return result, rows.Err()
}

func collectValues[T any](rows *sql.Rows, scan func(scanner) (T, error)) ([]T, error) {
    defer func() { _ = rows.Close() }()
    var result []T
    for rows.Next() {
        v, err := scan(rows)
        if err != nil {
            return nil, err
        }
        result = append(result, v)
    }
    return result, rows.Err()
}
```

Existing functions become one-liners:

```go
func collectFeeds(rows *sql.Rows) ([]*core.Feed, error)   { return collectRows(rows, scanFeed) }
func collectEntries(rows *sql.Rows) ([]*core.Entry, error) { return collectRows(rows, scanEntry) }
```

For `collectTags`, extract a `scanTag` function (currently inlined) and use `collectValues`.

### Affected Files

| File | Change |
|------|--------|
| `store/sqlite_scan.go` | Add `collectRows[T]`, `collectValues[T]`; extract `scanTag`; simplify existing functions |

### Effort: Small (1 hour)

---

## Proposal 4: Contextual Logging with `slog.With` (Go 1.21)

### Current Problem

Logger calls repeatedly pass the same key-value pairs. For example, in
`fetch_download.go`, `fetch_persist.go`, and `fetch_batch.go`, `"id", feed.ID`
appears in nearly every log statement:

```go
f.logger.Error("failed to fetch feed", "id", feed.ID, "url", feed.URL, "error", err)
p.logger.Info("feed fetched", "id", feed.ID, "title", feed.Title, "new_entries", inserted)
p.logger.Warn("failed to reset feed error", "id", feed.ID, "error", err)
```

### Proposed Change

Use `slog.With` to create sub-loggers with bound context:

```go
func (p *storeFeedPersister) persist(ctx context.Context, feed *Feed, doc *fetchedFeedDocument) (*persistedFeedEntries, error) {
    logger := p.logger.With("feed_id", feed.ID, "feed_title", feed.Title)
    // ...
    logger.Info("feed fetched", "new_entries", inserted)
    logger.Warn("failed to reset feed error", "error", err)
}
```

Additionally, implement `slog.LogValuer` on `FeedError` for structured error logging:

```go
func (e *FeedError) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Int64("feed_id", e.FeedID),
        slog.String("feed_url", e.FeedURL),
        slog.String("op", e.Op),
        slog.Any("cause", e.Err),
    )
}
```

### Affected Files

| File | Change |
|------|--------|
| `core/fetch_batch.go` | Use `slog.With("feed_id", feed.ID)` in worker |
| `core/fetch_download.go` | Sub-logger in `handleFeedDownloadError` |
| `core/fetch_persist.go` | Sub-logger in `persist` method |
| `core/errors.go` | Add `LogValue()` to `FeedError` |
| `core/feed.go` | Sub-logger in `AddFeed`, `RemoveFeed` |

### Effort: Small (1–2 hours)

---

## Proposal 5: Adopt `maps` Package (Go 1.21)

### Current Problem

`core/opml.go` in `ExportOPML` manually deduplicates tag names using a
`seenTags map[string]bool` and builds a sorted `tagNames []string`:

```go
taggedFeeds := make(map[string][]*Feed)
tagNames := make([]string, 0)
seenTags := make(map[string]bool)
for _, f := range feeds {
    for _, tag := range feedTags[f.ID] {
        taggedFeeds[tag.Name] = append(taggedFeeds[tag.Name], f)
        if !seenTags[tag.Name] {
            tagNames = append(tagNames, tag.Name)
            seenTags[tag.Name] = true
        }
    }
}
sort.Strings(tagNames)
```

### Proposed Change

Replace with `maps.Keys` + `slices.Sorted`:

```go
taggedFeeds := make(map[string][]*Feed)
for _, f := range feeds {
    for _, tag := range feedTags[f.ID] {
        taggedFeeds[tag.Name] = append(taggedFeeds[tag.Name], f)
    }
}
tagNames := slices.Sorted(maps.Keys(taggedFeeds))
```

This eliminates the `seenTags` map entirely and reduces the block by ~6 lines.

### Affected Files

| File | Change |
|------|--------|
| `core/opml.go` | Simplify tag name collection in `ExportOPML` |

### Effort: Tiny (15 min)

---

## Proposal 6: Generic Table Renderer in CLI Layer

### Current Problem

`cmd/render.go` has 4 table types (feeds, entries, entry links, tags, stats),
each with Plain and Styled variants — 10 functions total. Every function follows
the same structure: convert items to `[][]string`, then render with headers.

### Proposed Change

Introduce a generic table renderer:

```go
type tableDefinition[T any] struct {
    headers []string
    toRow   func(T) []string
}

func renderTable[T any](w io.Writer, items []T, def tableDefinition[T]) error {
    if useStyled(w) {
        return renderTableStyled(w, items, def)
    }
    return renderTablePlain(w, items, def)
}
```

Each model defines its columns declaratively:

```go
var feedTableDef = tableDefinition[*core.Feed]{
    headers: []string{"ID", "TITLE", "URL", "FETCHED", "STATUS"},
    toRow: func(f *core.Feed) []string {
        fetched := "-"
        if f.FetchedAt != nil {
            fetched = f.FetchedAt.Format("2006-01-02 15:04")
        }
        return []string{
            fmt.Sprintf("%d", f.ID), f.Title, f.URL, fetched, feedStatus(f.Disabled, f.ErrorCount),
        }
    },
}
```

### Affected Files

| File | Change |
|------|--------|
| `cmd/render.go` | Add generic `renderTable[T]`, replace 10 functions with 2 generic functions + 5 table definitions |

### Effort: Medium (2–3 hours)

---

## Proposal 7: Iterator Pattern with `iter.Seq2` (Go 1.23)

### Current Problem

Store methods like `ListEntries` always load all matching rows into a `[]*Entry`
slice. For large result sets, this is memory-inefficient.

### Proposed Change

Add an optional `EntryIterator` interface using `iter.Seq2`:

```go
// core/core.go
type EntryIterator interface {
    IterEntries(ctx context.Context, filter EntryFilter) iter.Seq2[*Entry, error]
}
```

```go
// store/sqlite_entries.go
func (s *SQLiteStore) IterEntries(ctx context.Context, filter EntryFilter) iter.Seq2[*core.Entry, error] {
    return func(yield func(*core.Entry, error) bool) {
        query, args := newEntryFilterQuery(filter).buildSelectEntries(filter)
        rows, err := s.executor(ctx).QueryContext(ctx, query, args...)
        if err != nil {
            yield(nil, fmt.Errorf("query entries: %w", err))
            return
        }
        defer rows.Close()
        for rows.Next() {
            entry, err := scanEntry(rows)
            if !yield(entry, err) {
                return
            }
            if err != nil {
                return
            }
        }
        if err := rows.Err(); err != nil {
            yield(nil, err)
        }
    }
}
```

Callers use range-over-func:

```go
for entry, err := range store.IterEntries(ctx, filter) {
    if err != nil { return err }
    // process one entry at a time
}
```

### Caveats

- Existing `ListEntries` is preserved for backward compatibility
- `EntryIterator` is an optional interface — use type assertion to detect support
- Most beneficial for CLI streaming output and large data exports

### Affected Files

| File | Change |
|------|--------|
| `core/core.go` | Add `EntryIterator` interface |
| `store/sqlite_entries.go` | Implement `IterEntries` |
| `store/sqlite_scan.go` | Add generic `iterRows[T]` helper |
| `cmd/entry_commands.go` | Optionally use iterator for large output |

### Effort: Medium (2–3 hours)

---

## Proposal 8: Expand Structured Error Types

### Current Problem

Only `FeedError` exists as a structured error type. Entry and store operations
use `fmt.Errorf` string wrapping only, making programmatic error inspection difficult.

### Proposed Change

Add `StoreError` to `core/errors.go`:

```go
type StoreError struct {
    Op    string // "add_entries", "list_feeds", etc.
    Table string // "feeds", "entries", "tags"
    Err   error
}

func (e *StoreError) Error() string {
    return fmt.Sprintf("%s %s: %v", e.Op, e.Table, e.Err)
}

func (e *StoreError) Unwrap() error { return e.Err }

func (e *StoreError) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("op", e.Op),
        slog.String("table", e.Table),
        slog.Any("cause", e.Err),
    )
}
```

### Affected Files

| File | Change |
|------|--------|
| `core/errors.go` | Add `StoreError` type with `LogValue()` |
| `store/sqlite_feed.go` | Wrap key operations with `StoreError` |
| `store/sqlite_entries.go` | Wrap key operations with `StoreError` |
| `cmd/` | Use `errors.As` for type-based error display |

### Effort: Medium (2–3 hours)

---

## Proposal 9: `context.AfterFunc` for Cleanup (Go 1.21)

### Current Problem

`fetchFeedsConcurrently` handles context cancellation via `select` + `ctx.Done()`
channel in its dispatch loop.

### Proposed Change

Could use `context.AfterFunc` for declarative cancellation cleanup. However, the
current implementation is already clean and correct. This proposal has low priority
and should only be applied if there is a clear benefit.

### Effort: Tiny (30 min)

### Note: Skip if current code is sufficient

---

## Proposal 10: `sync.OnceValue` / `sync.OnceValues` Consideration (Go 1.21)

### Current State

`core/entry_metadata.go`'s `cachedValue[T]` uses `sync.RWMutex` for manual management.

### Analysis

`sync.OnceValues` is suited for one-time initialization, but `cachedValue[T]` is
instantiated per-Entry and the parse function depends on entry fields. `sync.OnceValues`
cannot be used as a zero-value struct field (it requires initialization with a function),
making it unsuitable as a direct replacement.

**Conclusion:** The current `cachedValue[T]` implementation is well-designed and appropriate.
No change needed.

### Effort: None (no change required)

---

## Priority Matrix

| Proposal | Impact | Effort | Recommended Priority |
|----------|--------|--------|---------------------|
| 5. `maps` package adoption | Small | Tiny | **High** — instant win, zero risk |
| 1. `slices` package adoption | Small | Tiny | **High** — instant win, zero risk |
| 4. `slog.With` contextual logging | Medium | Small | **High** — readability + debuggability |
| 3. Generic store row collection | Medium | Small | **High** — reduces maintenance cost |
| 2. `errors.Join` error aggregation | Medium | Small | Medium — involves API change |
| 6. Generic table renderer | Medium | Medium | Medium — eliminates duplicate code |
| 8. Expanded structured error types | Medium | Medium | Medium — incremental adoption |
| 7. Iterator pattern (`iter.Seq2`) | High | Medium | Low — only needed at scale |
| 9. `context.AfterFunc` | Small | Tiny | Low — current code is fine |
| 10. `sync.OnceValues` | None | None | **Skip** — current impl is optimal |

## Recommended Execution Order

### Phase 1: Quick Wins (1–2 hours)
1. Proposal 5 — `maps.Keys` + `slices.Sorted`
2. Proposal 1 — `slices.Sort` / `slices.Clone`
3. Proposal 4 — `slog.With` introduction

### Phase 2: Structural Improvements (3–4 hours)
4. Proposal 3 — Generic `collectRows[T]`
5. Proposal 2 — `errors.Join` introduction
6. Proposal 6 — Generic table renderer

### Phase 3: Advanced Improvements (3–4 hours)
7. Proposal 8 — Expanded structured error types
8. Proposal 7 — Iterator pattern (when data volume warrants it)

## Prerequisites

- All proposals are independently implementable. No inter-dependencies exist
  (though Proposals 1 and 5 are efficient to do together).
- Proposal 2 changes the return signature behavior of `FetchAll`, so caller tests
  need updating.
- Proposal 7 introduces an optional interface, minimizing impact on existing code.

## Completion Criteria

- [ ] All existing tests pass for each proposal
- [ ] New tests added (especially for Proposals 2, 7, 8)
- [ ] `go vet ./...` and `golangci-lint run` are clean
- [ ] `go.mod` imports are tidy
