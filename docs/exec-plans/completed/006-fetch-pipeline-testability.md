# 006: Improve Fetch Pipeline Testability

## Overview

The fetch pipeline stages (download, parse, persist) are implemented as `Service` methods
that directly access `s.store`, `s.logger`, and `s.httpClient()`. Individual stages
cannot be unit-tested in isolation; the entire `Service` is required.
Separate stages behind interfaces to enable focused testing.

## Current Problem

### Pipeline Structure

```
fetchFeed (core/fetch.go)
  ├── downloadFeedDocument (core/fetch_download.go)  → s.httpClient(), s.store
  ├── parseFetchedEntries  (core/fetch_parse.go)     → free function ✓
  └── persistFetchedFeed   (core/fetch_persist.go)   → s.store, s.logger
```

`downloadFeedDocument` and `persistFetchedFeed` are Service methods with direct access.

### Current Testing

- `core/fetch_test.go` (598 lines) — E2E test with httptest server and full Service
- `core/fetch_internal_test.go` (47 lines) — only `filterFeedsForFetch` unit test
- Parse testing embedded in E2E test
- Download and persist have no isolated unit tests

### Problems

1. Testing download error handling (304, error recording) requires store + HTTP
2. Testing persist logic (cache headers, error reset) requires HTTP
3. Tests are slow (HTTP + SQLite)
4. Failure root cause is hard to pinpoint

## Improvement Plan

### Step 1: Isolate Parse Testing

`parseFetchedEntries` is already a free function. Create a dedicated test file:

```go
// core/fetch_parse_test.go
func TestParseFetchedEntries(t *testing.T) {
    entries, err := parseFetchedEntries(1, "https://example.com/feed", []byte(testRSS))
    // Field assertions
}
```

### Step 2: Extract Download Behind Interface

```go
// core/fetch_download.go
type feedDownloader interface {
    download(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error)
}

type httpFeedDownloader struct {
    client *http.Client
    store  feedDownloadStore
    logger *slog.Logger
}

// Minimal Store interface for download stage
type feedDownloadStore interface {
    UpdateFeedFetchedAt(ctx context.Context, id int64) error
    RecordFeedError(ctx context.Context, id int64, errMsg string) error
    UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
}
```

### Step 3: Extract Persist Behind Interface

```go
// core/fetch_persist.go
type feedPersister interface {
    persist(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error)
}

type storeFeedPersister struct {
    store  feedPersistStore
    logger *slog.Logger
}

type feedPersistStore interface {
    AddEntries(ctx context.Context, entries []*Entry) (int, error)
    UpdateFeedFetchedAt(ctx context.Context, id int64) error
    ResetFeedError(ctx context.Context, id int64) error
    UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
    ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error)
}
```

### Step 4: Assemble Pipeline in Fetcher

```go
type Fetcher struct {
    store      FeedStore
    downloader feedDownloader
    persister  feedPersister
    logger     *slog.Logger
}

func (f *Fetcher) fetchFeed(ctx context.Context, feed *Feed, notifier *fetchNotifier) ([]*Entry, error) {
    notifier.started(feed)

    if feed.Disabled {
        notifier.skipped(feed, FetchSkipDisabled)
        return nil, nil
    }

    document, skipped, err := f.downloader.download(ctx, feed)
    if err != nil { ... }
    if skipped { ... }

    result, err := f.persister.persist(ctx, feed, document)
    if err != nil { ... }

    return f.resolveEntries(ctx, feed, result)
}
```

### Step 5: Unit Test Each Stage

```go
// core/fetch_download_test.go
func TestDownloadReturnsNotModifiedOn304(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotModified)
    }))
    store := &stubDownloadStore{}
    downloader := &httpFeedDownloader{client: ts.Client(), store: store, logger: discardLogger}

    _, skipped, err := downloader.download(ctx, feed)
    // No store needed, HTTP-only test
}
```

```go
// core/fetch_persist_test.go
func TestPersistStoresEntriesAndUpdatesFetchedAt(t *testing.T) {
    store := &stubPersistStore{insertedCount: 2}
    persister := &storeFeedPersister{store: store, logger: discardLogger}

    result, err := persister.persist(ctx, feed, document)
    // No HTTP needed, store stub-only test
}
```

## Target Files

| File | Change |
|------|--------|
| `core/fetch_parse_test.go` | New: unit tests for parse stage |
| `core/fetch_download.go` | Extract `feedDownloader` interface |
| `core/fetch_download_test.go` | New: unit tests for download stage |
| `core/fetch_persist.go` | Extract `feedPersister` interface |
| `core/fetch_persist_test.go` | New: unit tests for persist stage |
| `core/fetch.go` | Restructure as pipeline assembler |
| `core/fetch_batch.go` | Use `Fetcher` type |

## Prerequisites

Can proceed in parallel with plan #005 (Service decomposition). `Fetcher` is one of the
domain types targeted by that plan.

## Risks

- New interfaces (`feedDownloader`, `feedPersister`) add internal complexity
  - Mitigated: internal interfaces only, no public API change
- Over-separation reduces clarity of the full pipeline flow
  - Mitigated: keep E2E test alongside unit tests

## Completion Criteria

- [ ] `parseFetchedEntries` has isolated unit tests
- [ ] Download stage testable without store or full Service
- [ ] Persist stage testable without HTTP or full Service
- [ ] E2E pipeline tests continue to pass
- [ ] Test execution time reduced
