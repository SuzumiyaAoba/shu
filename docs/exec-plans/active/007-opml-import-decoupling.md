# 007: Decouple OPML Import from HTTP Fetch

## Overview

OPML import calls `AddFeed`, which fetches each feed URL via HTTP.
This creates 1 HTTP request per feed during import (100 feeds = 100 requests).
Separate registration from validation to enable offline import and batch operations.

## Current Problem

### Import Call Flow

```
ImportOPML
  └── importOutline (N times)
        └── ensureFeed
              └── service.AddFeed
                    └── fetchBody (HTTP GET)  ← 1 request per feed
                    └── gofeed.Parse
                    └── store.AddFeed
```

### Problems

1. **Slow**: 100-feed OPML triggers 100 HTTP requests
2. **Fragile**: Single feed HTTP error skips that feed
3. **Offline impossible**: Cannot import without network
4. **Metadata duplication**: OPML has title, but HTTP re-fetches it
5. **Test friction**: OPML tests need httptest server

### AddFeed's Dual Responsibility

```go
func (s *Service) AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error) {
    body, err := s.fetchBody(ctx, url)          // Responsibility 1: HTTP fetch + validate
    parsed, err := fp.Parse(bytes.NewReader(body))
    // ...
    s.store.AddFeed(ctx, feed)                  // Responsibility 2: Persist
}
```

## Improvement Plan

### Step 1: Add AddFeedDirect — Registration Without Fetch

```go
// core/feed.go
func (s *Service) AddFeedDirect(ctx context.Context, feed *Feed) error {
    if feed.URL == "" {
        return fmt.Errorf("feed URL is required")
    }
    if err := s.store.AddFeed(ctx, feed); err != nil {
        return fmt.Errorf("store feed: %w", err)
    }
    s.logger.Info("feed added (direct)", "id", feed.ID, "title", feed.Title, "url", feed.URL)
    return nil
}
```

Keep existing `AddFeed` (CLI `shu add` benefits from validation).

### Step 2: Rebase OPML Import on AddFeedDirect

```go
func (i *opmlImporter) ensureFeed(ctx context.Context, url, title string) (*Feed, int, error) {
    feed := &core.Feed{
        URL:   url,
        Title: title,
    }
    err := i.service.AddFeedDirect(ctx, feed)
    if err == nil {
        if i.result != nil {
            i.result.AddedCount++
        }
        return feed, 1, nil
    }

    if errors.Is(err, ErrFeedAlreadyExists) {
        // Existing handling
    }
    // ...
}
```

### Step 3: Add Optional Background Fetch

```go
type OPMLImportResult struct {
    AddedCount  int
    ReusedCount int
    TaggedCount int
    Issues      []OPMLImportIssue
    AddedFeeds  []*Feed  // Added: newly registered feeds
}
```

CLI recommends post-import workflow: `shu import feed.opml && shu fetch`.
Or add `--fetch` flag for auto-fetch.

```go
// cmd/opml_commands.go
func newImportCmd(getService opmlServiceGetter) *cobra.Command {
    var fetchAfterImport bool
    // ...
    if fetchAfterImport {
        count, _ := svc.FetchAll(ctx)
        fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
    }
}
```

### Step 4: Simplify OPML Tests

```go
// core/opml_test.go — no httptest server needed
func TestImportOPML(t *testing.T) {
    svc := newTestService(t, nil)  // nil HTTP handler
    opmlData := `<opml>...<outline xmlUrl="https://example.com/feed" title="Test"/></opml>`

    result, err := svc.ImportOPMLDetailed(ctx, strings.NewReader(opmlData))
    // Fast, network-free test
}
```

## Target Files

| File | Change |
|------|--------|
| `core/feed.go` | Add `AddFeedDirect` method |
| `core/opml.go` | Base `ensureFeed` on `AddFeedDirect` |
| `core/opml_test.go` | Remove httptest dependency, speed up tests |
| `cmd/opml_commands.go` | Add `--fetch` flag (optional) |
| `cmd/services.go` | Ensure `opmlService` interface alignment |

## Phased Execution

1. Add `AddFeedDirect` (no existing code impact)
2. Switch OPML to `AddFeedDirect`
3. Update tests
4. Add `--fetch` flag (optional enhancement)

## Risks

- `AddFeedDirect` skips URL validation, allowing invalid URLs to be registered
  - **Intentional**: validation and registration are separate concerns
  - Invalid URLs fail when `shu fetch` runs (error_count increments)
- OPML with empty `title` or `text` attributes
  - Use OPML `title` > `text` > URL as fallback title

## Completion Criteria

- [ ] `AddFeedDirect` registers without HTTP requests
- [ ] OPML import completes without HTTP requests
- [ ] OPML tests run without httptest server
- [ ] Existing `shu add` command behavior unchanged
- [ ] Documentation shows `shu import` → `shu fetch` workflow
