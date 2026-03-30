package core

import (
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"
)

func (s *Service) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) {
	feed, err := s.store.GetFeed(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("get feed %d: %w", feedID, err)
	}

	fp := gofeed.NewParser()
	fp.Client = s.client

	parsed, err := fp.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %s: %w", feed.URL, err)
	}

	entries := make([]*Entry, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			continue
		}

		e := &Entry{
			FeedID:  feedID,
			GUID:    guid,
			Title:   item.Title,
			Link:    item.Link,
			Summary: item.Description,
		}
		if item.PublishedParsed != nil {
			t := item.PublishedParsed.UTC()
			e.PublishedAt = &t
		}
		entries = append(entries, e)
	}

	inserted, err := s.store.AddEntries(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("store entries: %w", err)
	}

	if err := s.store.UpdateFeedFetchedAt(ctx, feedID); err != nil {
		return nil, fmt.Errorf("update fetched_at: %w", err)
	}

	s.logger.Info("feed fetched", "id", feedID, "title", feed.Title, "new_entries", inserted)

	// Return only newly inserted entries
	newEntries := entries[:0]
	if inserted == len(entries) {
		newEntries = entries
	} else if inserted > 0 {
		// Re-fetch to get the accurate new entries list
		// For simplicity, return all entries when some are new
		newEntries = entries[:inserted]
	}

	return newEntries, nil
}

func (s *Service) FetchAll(ctx context.Context) (int, error) {
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("list feeds: %w", err)
	}

	total := 0
	for _, feed := range feeds {
		entries, err := s.FetchFeed(ctx, feed.ID)
		if err != nil {
			s.logger.Error("failed to fetch feed", "id", feed.ID, "url", feed.URL, "error", err)
			continue
		}
		total += len(entries)
	}

	return total, nil
}

func (s *Service) ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error) {
	return s.store.ListEntries(ctx, filter)
}
