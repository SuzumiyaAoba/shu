# 012: Concurrency Robustness

## Overview

Improve panic recovery in feed fetch workers, context propagation in migrations, and pre-allocation of error slices to increase the robustness of concurrent processing.

---

## Proposal 1: Panic Recovery in Fetch Workers

### Current Issue

The worker goroutine in `core/fetch_batch.go:73-99` lacks `recover()`.
If a panic occurs within a worker, `wg.Done()` is not called and `wg.Wait()` blocks forever (deadlock).

While panics from the gofeed parser or HTTP processing are unlikely in current code,
there is no defense against bugs in external libraries or runtime panics from out-of-memory conditions.

### Proposed Change

Add `recover` inside the worker function to convert panics to normal errors:

```go
worker := func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            errMu.Lock()
            fetchErrs = append(fetchErrs, fmt.Errorf("worker panic: %v", r))
            errMu.Unlock()
        }
    }()
    // ... existing worker logic
}
```

### Impact Scope

| File | Change |
|------|--------|
| `core/fetch_batch.go` | Add `recover()` to worker function |
| `core/fetch_batch_test.go` | Add test for panic recovery |

### Effort: Tiny (30 minutes)

---

## Proposal 2: Context Propagation in Migrations

### Current Issue

`store/sqlite_migrations.go:44` uses `context.Background()`:

```go
if _, err := provider.Up(context.Background()); err != nil {
```

This means migrations cannot be canceled even if the application shuts down during migration execution.

### Proposed Change

Modify the `runMigrations` signature to accept a context:

```go
func (s *SQLiteStore) runMigrations(ctx context.Context) error {
    // ...
    if _, err := provider.Up(ctx); err != nil {
```

Also add context to the calling functions `NewSQLiteStore` and `NewSQLiteStoreWithOptions`.

### Notes

- Changing `NewSQLiteStore` signature is a breaking change
- Since context can be passed from `cmd/root.go`'s `PersistentPreRunE`, the actual change is minimal
- goose's `provider.Up` accepts context.Context, so there are no API constraints

### Impact Scope

| File | Change |
|------|--------|
| `store/sqlite.go` | Add `context.Context` parameter to `NewSQLiteStore` and `NewSQLiteStoreWithOptions` |
| `store/sqlite_migrations.go` | Change to `runMigrations(ctx)` |
| `store/sqlite_test.go` | Update test helpers |
| `app/app.go` | Add context to `StoreOpener` type |
| `cmd/root.go` | Update to pass context |

### Effort: Small (1–2 hours)

---

## Proposal 3: Pre-allocate Fetch Error Slice

### Current Issue

`fetchErrs` is declared as a nil slice in `core/fetch_batch.go:70`,
and is extended incrementally with `append` when errors occur:

```go
var (
    fetchErrs  []error
)
```

When many feeds (1000+) produce frequent errors, slice reallocations happen frequently.

### Proposed Change

```go
fetchErrs := make([]error, 0, min(len(feeds), 64))
```

Pre-allocate with a cap of 64. This reduces memory waste when errors are few
and reduces allocations for moderate error counts.

### Impact Scope

| File | Change |
|------|--------|
| `core/fetch_batch.go` | Change `fetchErrs` declaration to pre-allocated |

### Effort: Tiny (10 minutes)

---

## Priority Matrix

| Proposal | Impact | Effort | Recommended Priority |
|----------|--------|--------|----------------------|
| 1. Panic recovery | High (prevent deadlock) | Tiny | **High** |
| 3. Pre-allocate error slice | Low (performance) | Tiny | **High** — Single-line change |
| 2. Migration context | Medium (graceful shutdown) | Small | Medium — Has breaking changes |

## Recommended Execution Order

1. Proposal 3 — Pre-allocate error slice (10 minutes)
2. Proposal 1 — Panic recovery (30 minutes)
3. Proposal 2 — Context propagation in migrations (1–2 hours)

## Completion Checklist

- [ ] All existing tests pass
- [ ] Proposal 1: Add test confirming no deadlock on panic
- [ ] Proposal 2: Confirm migration cancels when context is canceled
- [ ] `go vet ./...` and `golangci-lint run` clean
