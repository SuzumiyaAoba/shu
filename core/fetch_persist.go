package core

import (
	"context"
	"fmt"
	"net/http"
)

type persistedFeedEntries struct {
	entries  []*Entry
	inserted int
}

func (f *Fetcher) persistFetchedFeed(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error) {
	f.storeConditionalHeaders(ctx, feed.ID, document.headers)

	entries, err := parseFetchedEntries(feed.ID, feed.URL, document.body)
	if err != nil {
		return nil, err
	}

	inserted, err := f.store.AddEntries(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("store entries: %w", err)
	}
	if err := f.markFeedFetched(ctx, feed.ID); err != nil {
		return nil, err
	}

	if err := f.store.ResetFeedError(ctx, feed.ID); err != nil {
		f.logger.Warn("failed to reset feed error", "id", feed.ID, "error", err)
	}

	f.logger.Info("feed fetched", "id", feed.ID, "title", feed.Title, "new_entries", inserted)
	return &persistedFeedEntries{entries: entries, inserted: inserted}, nil
}

func (f *Fetcher) markFeedFetched(ctx context.Context, feedID int64) error {
	if err := f.store.UpdateFeedFetchedAt(ctx, feedID); err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

func (f *Fetcher) storeConditionalHeaders(ctx context.Context, feedID int64, headers http.Header) {
	if etag := headers.Get("ETag"); etag != "" || headers.Get("Last-Modified") != "" {
		if err := f.store.UpdateFeedCacheHeaders(ctx, feedID, headers.Get("ETag"), headers.Get("Last-Modified")); err != nil {
			f.logger.Warn("failed to update cache headers", "id", feedID, "error", err)
		}
	}
}

func (f *Fetcher) resolveFetchedEntries(ctx context.Context, feed *Feed, result *persistedFeedEntries) ([]*Entry, int) {
	if result.inserted == 0 {
		return nil, 0
	}
	if result.inserted == len(result.entries) {
		return result.entries, len(result.entries)
	}

	feedID := feed.ID
	newEntries, err := f.store.ListEntries(ctx, EntryFilter{FeedID: &feedID, Limit: result.inserted})
	if err != nil {
		f.logger.Warn("failed to retrieve newly inserted entries", "id", feed.ID, "error", err)
		return result.entries[:result.inserted], result.inserted
	}
	return newEntries, len(newEntries)
}
