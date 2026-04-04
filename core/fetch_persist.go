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

func (s *Service) persistFetchedFeed(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error) {
	s.storeConditionalHeaders(ctx, feed.ID, document.headers)

	entries, err := parseFetchedEntries(feed.ID, feed.URL, document.body)
	if err != nil {
		return nil, err
	}

	inserted, err := s.store.AddEntries(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("store entries: %w", err)
	}
	if err := s.markFeedFetched(ctx, feed.ID); err != nil {
		return nil, err
	}

	if err := s.store.ResetFeedError(ctx, feed.ID); err != nil {
		s.logger.Warn("failed to reset feed error", "id", feed.ID, "error", err)
	}

	s.logger.Info("feed fetched", "id", feed.ID, "title", feed.Title, "new_entries", inserted)
	return &persistedFeedEntries{entries: entries, inserted: inserted}, nil
}

func (s *Service) markFeedFetched(ctx context.Context, feedID int64) error {
	if err := s.store.UpdateFeedFetchedAt(ctx, feedID); err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

func (s *Service) storeConditionalHeaders(ctx context.Context, feedID int64, headers http.Header) {
	if etag := headers.Get("ETag"); etag != "" || headers.Get("Last-Modified") != "" {
		if err := s.store.UpdateFeedCacheHeaders(ctx, feedID, headers.Get("ETag"), headers.Get("Last-Modified")); err != nil {
			s.logger.Warn("failed to update cache headers", "id", feedID, "error", err)
		}
	}
}

func (s *Service) resolveFetchedEntries(ctx context.Context, feed *Feed, result *persistedFeedEntries) ([]*Entry, int) {
	if result.inserted == 0 {
		return nil, 0
	}
	if result.inserted == len(result.entries) {
		return result.entries, len(result.entries)
	}

	feedID := feed.ID
	newEntries, err := s.store.ListEntries(ctx, EntryFilter{FeedID: &feedID, Limit: result.inserted})
	if err != nil {
		s.logger.Warn("failed to retrieve newly inserted entries", "id", feed.ID, "error", err)
		return result.entries[:result.inserted], result.inserted
	}
	return newEntries, len(newEntries)
}
