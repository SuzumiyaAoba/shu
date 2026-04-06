package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SuzumiyaAoba/shu/model"
)

type feedPersister interface {
	persist(ctx context.Context, feed *model.Feed, document *fetchedDocument) (*persistedEntries, error)
}

type feedPersistStore interface {
	AddEntries(ctx context.Context, entries []*model.Entry) (int, error)
	UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	ResetFeedError(ctx context.Context, id int64) error
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
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

type persistedEntries struct {
	entries  []*model.Entry
	inserted int
}

func newStoreFeedPersister(store feedPersistStore, logger *slog.Logger) *storeFeedPersister {
	return &storeFeedPersister{
		store:  store,
		logger: ensureLogger(logger),
	}
}

func (p *storeFeedPersister) persist(ctx context.Context, feed *model.Feed, document *fetchedDocument) (*persistedEntries, error) {
	logger := p.logger.With("feed_id", feed.ID, "feed_title", feed.Title)

	// Cache headers are best-effort and do not need to be part of the transaction.
	storeConditionalHeaders(ctx, p.store, logger, feed.ID, document.headers)

	// Update the stored URL if the server issued a permanent redirect (301).
	if document.finalURL != "" && document.finalURL != feed.URL {
		logger.Info("feed URL redirected, updating", "old_url", feed.URL, "new_url", document.finalURL)
		newURL := document.finalURL
		if err := p.store.UpdateFeed(ctx, feed.ID, model.FeedUpdate{URL: &newURL}); err != nil {
			logger.Warn("failed to update redirected feed URL", "error", err)
		}
	}

	entries, err := parseFetchedEntries(feed.ID, feed.URL, document.body)
	if err != nil {
		return nil, err
	}

	inserted := 0
	if err := p.store.RunInTx(ctx, func(ctx context.Context) error {
		var err error
		inserted, err = p.store.AddEntries(ctx, entries)
		if err != nil {
			return fmt.Errorf("store entries: %w", err)
		}
		if err := markFeedFetched(ctx, p.store, feed.ID); err != nil {
			return err
		}
		if err := p.store.ResetFeedError(ctx, feed.ID); err != nil {
			logger.Warn("failed to reset feed error", "error", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	logger.Info("feed fetched", "new_entries", inserted)
	return &persistedEntries{entries: entries, inserted: inserted}, nil
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
			logger.Warn("failed to update cache headers", "error", err)
		}
	}
}
