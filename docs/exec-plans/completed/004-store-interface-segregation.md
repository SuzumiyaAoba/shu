# 004: Segregate Store Interface (Interface Segregation Principle)

## Overview

`core.Store` is a monolithic interface composing 6 sub-interfaces (~30 methods).
`Service` depends on all of them. Leverage existing sub-interfaces in testing
and simplify fake store implementations to reduce maintenance overhead.

## Current Problem

### Service Depends on Everything

```go
type Service struct {
    store Store  // Depends on all 30+ methods
}
```

### Test Fakes Bloat

```go
// app/test_helpers_test.go — implement 30+ no-op methods
type fakeStore struct{}
func (f *fakeStore) AddFeed(...) error                    { return nil }
func (f *fakeStore) GetFeed(...) (*core.Feed, error)      { return nil, nil }
// ... 28+ more stubs

// core/read_test.go — same 30+ no-op methods
type entryStateErrorStore struct{ err error }
// ... 30+ more stubs
```

Each new method requires changes to all fakes.

## Improvement Plan

### Step 1: Provide Common Base Fake Implementation

Existing sub-interfaces are already defined: `FeedStore`, `FeedHealthStore`,
`EntryStore`, `EntryStateStore`, `TagStore`, `MaintenanceStore`.

Create a shared base fake that implements all methods as no-ops.

### Step 2: Option A — Embed Base Fake in Tests

```go
// core/coretest/fake_store.go
type BaseFakeStore struct{}

func (f *BaseFakeStore) AddFeed(context.Context, *core.Feed) error                       { return nil }
func (f *BaseFakeStore) GetFeed(context.Context, int64) (*core.Feed, error)              { return nil, nil }
// ... all ~30 methods
```

### Step 3: Tests Override Only What They Need

```go
// core/read_test.go
type entryStateErrorStore struct {
    *coretest.BaseFakeStore
    err error
}

func (s *entryStateErrorStore) MarkEntryRead(ctx context.Context, id int64) error {
    return s.err
}
```

Only 1-2 methods overridden per test fake, not 30+.

### Step 4: Share BaseFakeStore Across Packages

Make `BaseFakeStore` available to both `app` and `core` test packages:

```
core/
  coretest/
    fake_store.go  // BaseFakeStore — implements all methods as no-op
```

## Target Files

| File | Change |
|------|--------|
| `core/coretest/fake_store.go` | New: `BaseFakeStore` with all methods as no-op |
| `app/test_helpers_test.go` | Replace `fakeStore` implementation with embed of `BaseFakeStore` |
| `core/read_test.go` | `entryStateErrorStore` embeds `BaseFakeStore`, overrides only needed methods |
| Other test files | Similar simplification |

## Risk Mitigation

- Keep `coretest` internal (avoid external dependency)
- Document that test fakes must inherit from `BaseFakeStore`
- Enforce in code review that missing method overrides are caught by compile errors

## Completion Criteria

- [x] `BaseFakeStore` exists and implements all ~30 Store methods as no-op
- [x] All test fakes embed `BaseFakeStore` and override only what they need
- [x] New method additions only require changes to `BaseFakeStore`
- [x] All existing tests pass
