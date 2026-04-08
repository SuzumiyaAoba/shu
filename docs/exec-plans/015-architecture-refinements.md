# 015: Architecture Refinements

## Overview

Address five architectural improvement opportunities identified in the current
codebase. Each proposal is independent and can be implemented in any order.

---

## Proposal 1: Extract EntryMetadataCache from Domain Model

### Current Problem

`model.Entry` embeds `EntryMetadataCache` which contains `sync.RWMutex` fields.
This mixes caching/concurrency concerns into a pure domain model:

```go
// model/model.go
type Entry struct {
    // ... domain fields ...
    metadataCache EntryMetadataCache `json:"-"` // sync.RWMutex inside
}
```

Issues:
- Domain model carries infrastructure concerns (mutexes, lazy evaluation)
- `CachedValue[T]` contains `sync.RWMutex`, which is not safe to copy —
  but `Entry` is a value-friendly struct elsewhere in the codebase
- Cache is invalidated on JSON round-trip (tagged `json:"-"`)

### Proposed Changes

Move metadata parsing to a standalone helper package or a top-level function
set, removing the cache from `Entry` entirely.

**Option A: Stateless parse functions (simplest)**

Replace `Entry.ParseCategories()` etc. with standalone functions that parse on
every call. JSON fields are small (typically < 1 KB), so the parse cost is
negligible for CLI usage:

```go
// model/entry_metadata.go

func ParseCategories(e *Entry) ([]EntryCategory, error) {
    return ParseRawSlice[EntryCategory](e.Categories, "categories")
}

func ParseEnclosures(e *Entry) ([]EntryEnclosure, error) {
    return ParseRawSlice[EntryEnclosure](e.Enclosures, "enclosures")
}

// ... same for Authors, Links, Contributors, Source
```

Remove `EntryMetadataCache`, `CachedValue[T]`, and the `metadataCache` field
from `Entry`.

**Option B: External cache wrapper (if caching is needed later)**

If profiling shows parse cost matters, introduce a wrapper in the consuming
layer:

```go
// core/entry_view.go
type EntryView struct {
    *model.Entry
    cache model.EntryMetadataCache
}
```

This keeps `Entry` clean and moves caching to the layer that needs it.

### Recommended Approach

Option A. The CLI processes entries sequentially and each field is parsed at
most once per entry per command invocation. The mutex overhead is larger than
the parse savings for payloads this small.

### Migration Steps

1. Convert `Entry.ParseCategories()` → `model.ParseCategories(e *Entry)`
   (repeat for all six Parse* methods)
2. Remove `metadataCache` field from `Entry`
3. Remove `EntryMetadataCache` and `CachedValue[T]` types
4. Update all callers (grep for `.ParseCategories(`, `.ParseEnclosures(`, etc.)
5. Remove `model/entry_metadata_test.go` cache-specific tests; keep parse tests

### Files Changed

- `model/model.go` — remove `metadataCache` field
- `model/entry_metadata.go` — replace methods with functions, remove cache types
- `model/entry_metadata_test.go` — update tests
- `cmd/render.go`, `cmd/output.go` — update call sites
- `core/fetch/parse.go` — update if metadata is parsed during fetch

---

## Proposal 2: Decouple OPMLHandler from Manager Interfaces

### Current Problem

`OPMLHandler` depends on `opmlFeedAdder` and `opmlTagAdder`, which mirror
`FeedManager` and `TagManager` methods. This creates implicit coupling between
peer managers:

```go
// core/opml.go
type OPMLHandler struct {
    feedStore FeedStore       // store layer ✓
    tagStore  TagStore        // store layer ✓
    feeds     opmlFeedAdder   // manager layer — tight coupling
    tags      opmlTagAdder    // manager layer — tight coupling
    logger    *slog.Logger
}
```

If `FeedManager.AddFeed` or `FeedManager.AddFeedDirect` signatures change,
`OPMLHandler` breaks even though it has no direct import.

### Proposed Changes

Remove the manager-shaped dependencies. OPMLHandler should operate at the store
level for import, and delegate orchestration (validation, fetch-on-add) to the
caller.

**Step 1: Simplify OPMLHandler to store-only dependencies**

```go
type OPMLHandler struct {
    feedStore FeedStore
    tagStore  TagStore
    txRunner  TxRunner
    logger    *slog.Logger
}
```

**Step 2: Import uses store directly**

For import, `OPMLHandler` calls `feedStore.AddFeed` (insert) and
`tagStore.AddTag` (tag) directly. Feed URL validation moves to a shared
utility that both `FeedManager` and `OPMLHandler` can call.

```go
func (i *opmlImporter) ensureFeed(ctx context.Context, url, title string) (*model.Feed, int, error) {
    if err := ValidateFeedURL(url, false); err != nil {
        // record issue, skip
    }
    feed := &model.Feed{URL: url, Title: title}
    if err := i.handler.feedStore.AddFeed(ctx, feed); err != nil {
        if errors.Is(err, model.ErrFeedAlreadyExists) {
            return i.handler.feedStore.GetFeedByURL(ctx, url)
        }
        return nil, 0, err
    }
    return feed, 1, nil
}
```

**Step 3: Update Service wiring**

```go
// core/core.go — New()
svc := &Service{
    // ...
    opml: NewOPMLHandler(store, store, store, logger), // feedStore, tagStore, txRunner
}
```

### Files Changed

- `core/opml.go` — remove opmlFeedAdder/opmlTagAdder, use store interfaces
- `core/core.go` — update NewOPMLHandler call in New()
- `core/opml_test.go` — simplify test doubles

---

## Proposal 3: Consistent Command-Layer Error Wrapping

### Current Problem

Commands in `cmd/` bubble errors without adding CLI-level context:

```go
// cmd/feed_commands.go (typical pattern)
feed, err := svc.AddFeed(cmd.Context(), url, title)
if err != nil {
    return err  // no command context
}
```

When a store error propagates, the user sees a raw chain like:
`store feed: UNIQUE constraint failed: feeds.url` without knowing which
command or argument caused it.

### Proposed Changes

Wrap errors at the command boundary with the command name and key arguments:

```go
feed, err := svc.AddFeed(cmd.Context(), url, title)
if err != nil {
    return fmt.Errorf("add feed %q: %w", url, err)
}
```

### Implementation Guidelines

1. **Wrap at the outermost RunE boundary**, not in helper functions
2. **Include the primary identifier** (feed URL, entry ID, tag name, file path)
3. **Do not double-wrap** — if the domain layer already includes the identifier
   (e.g., `FeedError` with FeedURL), just wrap with the command name
4. **User-facing errors** — for known error types (`ErrFeedAlreadyExists`,
   `ErrFeedNotFound`), consider presenting a friendly message instead of the
   raw error chain

### Audit Checklist

Review each command file and ensure RunE functions wrap errors:

- [ ] `cmd/feed_commands.go` — add, list, remove, update, enable, disable, discover
- [ ] `cmd/entry_commands.go` — list, get, read, unread, star, unstar, search, duplicates
- [ ] `cmd/tag_commands.go` — add, remove, list
- [ ] `cmd/opml_commands.go` — import, export
- [ ] `cmd/maintenance_commands.go` — stats, cleanup, fetch

### Files Changed

- `cmd/feed_commands.go`
- `cmd/entry_commands.go`
- `cmd/tag_commands.go`
- `cmd/opml_commands.go`
- `cmd/maintenance_commands.go`

---

## Proposal 4: Remove Empty store/store.go

### Current Problem

`store/store.go` contains only a package-level doc comment and no code. The
comment references `core.Store` and `core.Feed`/`core.Entry` which now live in
the `model` package, making the comment outdated.

```go
// Package store provides the persistence layer for the shu RSS aggregator.
//
// It provides a concrete SQLite implementation via [SQLiteStore], which
// satisfies the [core.Store] interface. The package imports [core] only for
// model types ([core.Feed], [core.Entry], [core.EntryFilter]) and must never
// depend on the cmd package.
package store
```

### Proposed Changes

**Option A (preferred):** Move the package doc comment to `sqlite.go` (the
primary implementation file) and delete `store.go`.

**Option B:** Update the comment to reference `model.Feed`, `model.Entry`, etc.
and keep the file as the canonical doc location.

### Files Changed

- `store/store.go` — delete
- `store/sqlite.go` — add package doc comment (if Option A)

---

## Proposal 5: Interface-Based Service for Testability

### Current Problem

`core.Service` is a concrete struct. While `cmd/services.go` defines
per-command interfaces (`feedService`, `entryService`, etc.) for partial
decoupling, the full `*core.Service` is still threaded through the
composition root:

```go
// cmd/root.go
func newRootCmd(injected *core.Service) *cobra.Command { ... }
```

This makes it difficult to:
- Mock the entire service in integration tests
- Swap implementations (e.g., a read-only service for `shu serve`)
- Test commands in isolation without constructing a full Service

### Proposed Changes

This is the lowest-priority proposal because the per-command interfaces in
`cmd/services.go` already provide good decoupling. Consider this only if
the project grows to need full service mocking or alternative implementations.

**Step 1: Define a top-level service interface**

```go
// core/core.go
type Services interface {
    feedService
    entryService
    tagService
    opmlService
    maintenanceService
    discoveryService
    fetchService
}
```

Where each sub-interface groups related methods (these already exist implicitly
in `cmd/services.go` — promote them to `core/`).

**Step 2: Service implements Services**

```go
var _ Services = (*Service)(nil) // compile-time check
```

**Step 3: Update composition root**

```go
func newRootCmd(injected Services) *cobra.Command { ... }
```

### Trade-offs

- **Pro:** Full service mocking in tests, alternative implementations possible
- **Pro:** Compile-time verification that Service satisfies the contract
- **Con:** ~40 methods in the interface — large surface area to maintain
- **Con:** Current per-command interfaces already provide most of the benefit

### Recommendation

Defer this proposal unless a concrete need arises (e.g., a `shu serve` HTTP
mode or a Lambda handler that needs a different Service implementation). The
per-command interface pattern in `cmd/services.go` is sufficient for current
testing needs.

### Files Changed (if implemented)

- `core/core.go` — add Services interface
- `cmd/root.go` — accept Services instead of *Service
- `cmd/services.go` — remove redundant interfaces or re-export from core
- `core/coretest/` — add mock service implementation

---

## Priority and Sequencing

| # | Proposal | Impact | Effort | Priority |
|---|----------|--------|--------|----------|
| 1 | Extract EntryMetadataCache | Medium — cleaner domain model | Low | High |
| 2 | Decouple OPMLHandler | Medium — reduces coupling | Low | High |
| 3 | Consistent error wrapping | Medium — better UX | Low | Medium |
| 4 | Remove empty store.go | Low — cleanup | Trivial | Medium |
| 5 | Interface-based Service | Low (current) — future-proofing | Medium | Low |

Recommended order: 4 → 1 → 2 → 3 → 5 (if needed)

Proposal 4 is trivial and can be done immediately. Proposals 1 and 2 are
independent and can be implemented in parallel. Proposal 3 is a mechanical
audit. Proposal 5 should be deferred unless a concrete use case emerges.
