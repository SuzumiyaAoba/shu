# 005: Decompose Service into Domain Types

## Overview

`core.Service` has ~30 public methods across 9 unrelated domains (feeds, fetching, entries,
read/star state, tags, OPML, maintenance, dead feeds, discovery).
Decompose into focused domain types while keeping `Service` as a Facade for backward compatibility.

## Current Problem

`Service` mixes responsibilities:

1. **Feed management** — AddFeed, ListFeeds, GetFeed, RemoveFeed, UpdateFeed, EnableFeed, DisableFeed
2. **Fetching** — FetchFeed, FetchAll
3. **Entry queries** — ListEntries, ListEntriesPage, SearchEntries, GetEntry, FindDuplicateEntries
4. **Entry state** — MarkEntryRead/Unread, StarEntry/Unstar (8 methods)
5. **Tags** — AddTag, RemoveTag, ListTags, ListAllTags, ListFeedTags, ListFeedsByTag
6. **OPML** — ImportOPML, ImportOPMLDetailed, ExportOPML
7. **Maintenance** — FeedStatsAll, CleanupEntries
8. **Dead feeds** — ListDeadFeeds, RemoveDeadFeeds
9. **Discovery** — DiscoverFeeds

These unrelated domains share one struct with no cohesion.

## Improvement Plan

### Step 1: Define Domain Types

```go
// core/fetcher.go
type Fetcher struct {
    store  fetchStore
    logger *slog.Logger
    client *http.Client
}

func (f *Fetcher) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) { ... }
func (f *Fetcher) FetchAll(ctx context.Context) (int, error) { ... }
```

```go
// core/entry_queries.go
type EntryQueries struct {
    store entryQueryStore
}

func (q *EntryQueries) ListEntriesPage(ctx context.Context, filter EntryFilter) (*EntryPage, error) { ... }
```

### Step 2: Domain Type Allocation

| Type | Responsibility | Store Interfaces |
|------|-----------------|------------------|
| `FeedManager` | Feed CRUD, Enable/Disable | `FeedStore`, `FeedHealthStore` |
| `Fetcher` | Download, parse, persist | `FeedStore`, `FeedHealthStore`, `EntryStore` |
| `EntryQueries` | Search, pagination | `EntryStore` |
| `EntryStateManager` | Read/Unread, Star/Unstar | `EntryStateStore` |
| `TagManager` | Tag CRUD | `TagStore` |
| `OPMLHandler` | Import/Export | `FeedStore`, `TagStore` (+ `FeedManager`) |
| `MaintenanceOps` | Stats, cleanup, dead feeds | `MaintenanceStore`, `FeedStore`, `FeedHealthStore` |
| `FeedDiscovery` | URL discovery | (HTTP only) |

### Step 3: Convert Service to Facade

```go
type Service struct {
    feeds       *FeedManager
    fetcher     *Fetcher
    entries     *EntryQueries
    entryState  *EntryStateManager
    tags        *TagManager
    opml        *OPMLHandler
    maintenance *MaintenanceOps
    discovery   *FeedDiscovery
}

// Backward-compatible delegation methods
func (s *Service) AddFeed(ctx context.Context, url, title string) (*Feed, error) {
    return s.feeds.AddFeed(ctx, url, title)
}
```

### Step 4: Phased Migration

1. Extract `Fetcher` first (most complex, benefits most from isolation)
2. Extract `EntryStateManager` (pure delegation, easy)
3. Extract `FeedDiscovery` (no Store dependency, independent)
4. Migrate remaining types

## Prerequisites

Plan #004 (Store interface segregation) recommended—each domain type depends on a specific sub-interface,
making test fakes more manageable.

## Target Files

| File | Change |
|------|--------|
| `core/fetcher.go` | New: `Fetcher` type |
| `core/feed_manager.go` | New or rename `core/feed.go` → `FeedManager` |
| `core/entry_queries.go` | New: `EntryQueries` type |
| `core/entry_state.go` | Rename to `EntryStateManager` |
| `core/tag_manager.go` | Extract to `TagManager` |
| `core/opml_handler.go` | Extract to `OPMLHandler` |
| `core/maintenance_ops.go` | Extract to `MaintenanceOps` |
| `core/feed_discovery.go` | Extract to `FeedDiscovery` |
| `core/core.go` | Convert `Service` to Facade |
| All tests | Migrate to domain type unit tests |

## Risks

- Over-decomposition creates indirection without benefit (each type must have 3+ methods)
- OPML depends on `FeedManager`, creating a small dependency graph
  - Mitigated: use interface injection
- `cmd/services.go` interfaces may need eventual alignment with domain types

## Completion Criteria

- [ ] `Fetcher` independently instantiable and testable
- [ ] Each domain type depends on minimum necessary Store sub-interfaces
- [ ] `Service` methods are single-line delegations only
- [ ] All existing tests pass
- [ ] `cmd` layer requires no changes (Facade maintains API)
