package core

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type feedPersister interface {
	persist(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error)
}

type feedPersistStore interface {
	AddEntries(ctx context.Context, entries []*Entry) (int, error)
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	ResetFeedError(ctx context.Context, id int64) error
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
}

type feedFetchedMarker interface {
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
}

type feedCacheHeaderStore interface {
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
}

type storeFeedPersister struct {
	store  feedPersistStore
	logger *slog.Logger
}

type persistedFeedEntries struct {
	entries  []*Entry
	inserted int
}

func newStoreFeedPersister(store feedPersistStore, logger *slog.Logger) *storeFeedPersister {
	return &storeFeedPersister{
		store:  store,
		logger: normalizeLogger(logger),
	}
}

func (p *storeFeedPersister) persist(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error) {
	storeConditionalHeaders(ctx, p.store, p.logger, feed.ID, document.headers)

	entries, err := parseFetchedEntries(feed.ID, feed.URL, document.body)
	if err != nil {
		return nil, err
	}

	inserted, err := p.store.AddEntries(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("store entries: %w", err)
	}
	if err := markFeedFetched(ctx, p.store, feed.ID); err != nil {
		return nil, err
	}

	if err := p.store.ResetFeedError(ctx, feed.ID); err != nil {
		p.logger.Warn("failed to reset feed error", "id", feed.ID, "error", err)
	}

	p.logger.Info("feed fetched", "id", feed.ID, "title", feed.Title, "new_entries", inserted)
	return &persistedFeedEntries{entries: entries, inserted: inserted}, nil
}

func markFeedFetched(ctx context.Context, store feedFetchedMarker, feedID int64) error {
	if err := store.UpdateFeedFetchedAt(ctx, feedID); err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

func storeConditionalHeaders(ctx context.Context, store feedCacheHeaderStore, logger *slog.Logger, feedID int64, headers http.Header) {
	if etag := headers.Get("ETag"); etag != "" || headers.Get("Last-Modified") != "" {
		if err := store.UpdateFeedCacheHeaders(ctx, feedID, headers.Get("ETag"), headers.Get("Last-Modified")); err != nil {
			logger.Warn("failed to update cache headers", "id", feedID, "error", err)
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
