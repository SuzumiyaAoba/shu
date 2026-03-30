package core

import (
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"
)

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID, then
// stores any new entries that are not already in the database.
//
// The method performs the following steps:
//  1. Looks up the feed URL from the store by feedID.
//  2. Fetches and parses the feed document via HTTP.
//  3. Converts each feed item into an [Entry]. If an item lacks a GUID, its
//     Link is used as the deduplication key. Items with neither GUID nor Link
//     are silently skipped.
//  4. Inserts entries into the store. Duplicates (same feed_id + GUID) are
//     automatically ignored by the store's UNIQUE constraint.
//  5. Updates the feed's FetchedAt timestamp.
//
// It returns a slice containing only the newly inserted entries. An error is
// returned if the feed does not exist, the HTTP request fails, or the store
// operation fails.
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

	// Return only newly inserted entries.
	newEntries := entries[:0]
	if inserted == len(entries) {
		newEntries = entries
	} else if inserted > 0 {
		newEntries = entries[:inserted]
	}

	return newEntries, nil
}

// FetchAll fetches every registered feed sequentially and returns the total
// number of new entries stored across all feeds.
//
// If an individual feed fails to fetch (network error, parse error, etc.), the
// error is logged and the method continues with the remaining feeds. This
// ensures that a single broken feed does not block updates for others.
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

// ListEntries retrieves stored entries matching the given filter criteria.
// Results are ordered by fetched_at descending (newest first). It delegates
// directly to the store without additional business logic.
func (s *Service) ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error) {
	return s.store.ListEntries(ctx, filter)
}
