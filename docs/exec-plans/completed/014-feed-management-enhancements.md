# 014: Feed Management Enhancements

## Overview

Enhance feed management functionality with HTTP redirect URL auto-update,
entry date range filtering, and feed discovery fallback strategies.

---

## Proposal 1: Automatic Feed URL Update on HTTP 301 Redirect

### Current Problem

When a feed provider changes the URL and returns 301 (Moved Permanently),
`go-retryablehttp` automatically follows the redirect. However, the feed URL
in the database remains old, causing a redirect on every fetch.

Some feed servers provide redirects indefinitely, but there is a risk that
the old URL will be deleted in the future.

### Proposed Changes

Track the final response URL in `fetch_download.go` and auto-update the feed
URL on 301 redirect:

```go
type fetchedFeedDocument struct {
    body       []byte
    headers    http.Header
    finalURL   string // Final URL after redirect
    redirected bool   // Whether a 301 redirect occurred
}
```

Modify `fetchBodyConditional` to return the response's `Request.URL` (final URL
after redirect):

```go
func fetchBodyConditional(ctx context.Context, client *http.Client, url, etag, lastModified string) ([]byte, http.Header, string, error) {
    // ...
    finalURL := resp.Request.URL.String()
    return body, resp.Header, finalURL, nil
}
```

On persist, log URL changes and update via store's `UpdateFeed`:

```go
if document.redirected && document.finalURL != feed.URL {
    logger.Info("feed URL redirected, updating", "old_url", feed.URL, "new_url", document.finalURL)
    if err := p.store.UpdateFeed(ctx, feed.ID, FeedUpdate{URL: &document.finalURL}); err != nil {
        logger.Warn("failed to update redirected feed URL", "error", err)
    }
}
```

### Implementation Notes

- Only process 301 redirects (ignore 302/307 as temporary)
- Use `CheckRedirect` to verify redirect chain and detect 301 only
- URL update is best-effort (fetch succeeds even if update fails)
- Deduplication: skip if the redirect target URL already exists as another feed

### Affected Files

| File | Change |
|------|--------|
| `core/fetch_download.go` | Add final URL to `fetchBodyConditional` return values and detect redirects |
| `core/fetch_persist.go` | Add URL update logic on redirect detection |
| `core/fetch_download_test.go` | Add test for URL update on 301 redirect |

### Effort: Medium (2–3 hours)

---

## Proposal 2: Entry Date Range Filtering

### Current Problem

The `EntryFilter` in `core/model.go` supports filtering by `FeedID`, `UnreadOnly`,
`StarredOnly`, and `Tag`, but lacks date range filtering
(`--published-after`, `--published-before`).

Users cannot perform time-series operations like "show me this week's articles"
or "check my starred articles from last month".

### Proposed Changes

Add date range fields to `EntryFilter`:

```go
type EntryFilter struct {
    // ... existing fields
    PublishedAfter  *time.Time `json:"published_after"`
    PublishedBefore *time.Time `json:"published_before"`
}
```

Add filter conditions to `newEntryFilterQuery` in `store/sqlite_entries.go`:

```go
if filter.PublishedAfter != nil {
    query.add(`published_at >= ?`, filter.PublishedAfter.Format(time.RFC3339))
}
if filter.PublishedBefore != nil {
    query.add(`published_at < ?`, filter.PublishedBefore.Format(time.RFC3339))
}
```

Add CLI flags:

```go
entriesCmd.Flags().StringVar(&publishedAfter, "published-after", "", "filter entries published after this date (YYYY-MM-DD)")
entriesCmd.Flags().StringVar(&publishedBefore, "published-before", "", "filter entries published before this date (YYYY-MM-DD)")
```

### Affected Files

| File | Change |
|------|--------|
| `core/model.go` | Add `PublishedAfter`/`PublishedBefore` to `EntryFilter` |
| `store/sqlite_entries.go` | Add date conditions to `newEntryFilterQuery` |
| `cmd/entry_commands.go` | Add `--published-after`/`--published-before` flags to `entries` command |
| `store/sqlite_entries_test.go` | Add date range filter tests |

### Effort: Small (1–2 hours)

---

## Proposal 3: Feed Discovery Fallback Strategy

### Current Problem

`DiscoverFeeds` in `core/discover.go` only parses `<link rel="alternate">`
tags in HTML pages. While many sites have this tag, some (especially those
using static site generators) do not.

### Proposed Changes

Add fallback strategies when no `<link rel="alternate">` is found:

1. **Explore Common Paths**: Check for common feed paths (`/feed`, `/feed.xml`,
   `/rss`, `/rss.xml`, `/atom.xml`, `/index.xml`, `/feed/atom`, `/feed/rss`)
   with HEAD requests
2. **Detect JSON Feeds**: In addition to existing `application/feed+json`
   `<link>` detection, also check `/.well-known/feed` path

```go
var commonFeedPaths = []string{
    "/feed", "/feed.xml", "/rss", "/rss.xml",
    "/atom.xml", "/index.xml", "/feed/atom", "/feed/rss",
}

func (d *FeedDiscovery) discoverByCommonPaths(ctx context.Context, baseURL string) []string {
    var found []string
    for _, path := range commonFeedPaths {
        candidate := resolveURL(baseURL, path)
        if d.isFeedURL(ctx, candidate) {
            found = append(found, candidate)
        }
    }
    return found
}

func (d *FeedDiscovery) isFeedURL(ctx context.Context, url string) bool {
    req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
    if err != nil {
        return false
    }
    resp, err := d.client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return false
    }
    ct := resp.Header.Get("Content-Type")
    return strings.Contains(ct, "xml") || strings.Contains(ct, "rss") ||
        strings.Contains(ct, "atom") || strings.Contains(ct, "feed+json")
}
```

### Implementation Notes

- Run fallback discovery only if `<link>` parsing returns no results
- Use HEAD requests to avoid downloading body
- For servers without explicit Content-Type, consider GET-first few bytes
  to detect XML/JSON as an optional enhancement
- Parallelize common path checks (many paths to check, sequential would be slow)

### Affected Files

| File | Change |
|------|--------|
| `core/discover.go` | Add `discoverByCommonPaths` method and invoke as fallback from `DiscoverFeeds` |
| `core/discover_test.go` | Add fallback discovery tests |

### Effort: Medium (2–3 hours)

---

## Proposal 4: Expose per-feed fetch interval via CLI

### Current Problem

The `FetchIntervalSec` field exists in `core/model.go:55` and per-feed
interval skip logic is already implemented in `fetch_batch.go:56-61`,
but there's no CLI way to set this value.

The `FeedUpdate` in `core/model.go:166-169` only includes `Title` and `URL`,
not `FetchIntervalSec`.

### Proposed Changes

1. Add `FetchIntervalSec` to `FeedUpdate`:

```go
type FeedUpdate struct {
    Title            *string `json:"title"`
    URL              *string `json:"url"`
    FetchIntervalSec *int    `json:"fetch_interval_sec"`
}
```

2. Update `UpdateFeed` in `store/sqlite_feed.go` to handle the new column

3. Add flag to `update` command in `cmd/feed_commands.go`:

```go
updateCmd.Flags().DurationVar(&fetchInterval, "fetch-interval", 0,
    "per-feed fetch interval (e.g. 1h, 30m); 0 uses global default")
```

### Affected Files

| File | Change |
|------|--------|
| `core/model.go` | Add `FetchIntervalSec` to `FeedUpdate` |
| `store/sqlite_feed.go` | Update `UpdateFeed` to handle `FetchIntervalSec` column |
| `cmd/feed_commands.go` | Add `--fetch-interval` flag to `update` command |
| `store/sqlite_feed_test.go` | Add test for `UpdateFeed` with interval |

### Effort: Small (1 hour)

---

## Priority Matrix

| Proposal | Impact | Effort | Recommended Priority |
|----------|--------|--------|----------------------|
| 4. Expose per-feed interval CLI | Medium (reuse existing logic) | Small | **High** — Missing UI for existing feature |
| 2. Date range filtering | Medium (UX improvement) | Small | **High** |
| 1. Auto-update 301 redirect URL | Medium (operational efficiency) | Medium | Medium |
| 3. Feed discovery fallback | Medium (improve discovery rate) | Medium | Low — nice to have |

## Recommended Execution Order

### Phase 1: Complete Existing Features (2–3 hours)
1. Proposal 4 — Expose per-feed interval CLI
2. Proposal 2 — Date range filtering

### Phase 2: Enhance Feed Management (4–6 hours)
3. Proposal 1 — Auto-update 301 redirect URL
4. Proposal 3 — Feed discovery fallback

## Completion Criteria

- [ ] All existing tests pass
- [ ] Add tests for each Proposal
- [ ] `golangci-lint run` clean
- [ ] CLI help messages are consistent
