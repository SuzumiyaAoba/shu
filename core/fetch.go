package core

import (
	"context"
	"fmt"
)

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID, then
// stores any new entries that are not already in the database.
//
// For Atom feeds, the response body is parsed twice: once with gofeed's
// universal parser for common fields, and once with the Atom-specific parser
// to capture fields lost in translation (Contributors, Rights, Source,
// Author URIs, structured Categories, and full Link metadata).
//
// It returns a slice containing only the newly inserted entries.
func (s *Service) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) {
	return s.FetchFeedWithObserver(ctx, feedID, nil)
}

// FetchFeedWithObserver downloads and parses a single feed while emitting
// structured progress events to observer.
func (s *Service) FetchFeedWithObserver(ctx context.Context, feedID int64, observer FetchObserver) ([]*Entry, error) {
	notifier := newFetchNotifier(observer)
	return s.fetchFeedByID(ctx, feedID, notifier)
}

func (s *Service) fetchFeedByID(ctx context.Context, feedID int64, notifier *fetchNotifier) ([]*Entry, error) {
	feed, err := s.store.GetFeed(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("get feed %d: %w", feedID, err)
	}

	return s.fetchFeed(ctx, feed, notifier)
}

func (s *Service) fetchFeed(ctx context.Context, feed *Feed, notifier *fetchNotifier) ([]*Entry, error) {
	notifier.started(feed)

	if feed.Disabled {
		s.logger.Warn("feed disabled, skipping", "id", feed.ID, "title", feed.Title)
		notifier.skipped(feed, FetchSkipDisabled)
		return nil, nil
	}

	document, skipped, err := s.downloadFeedDocument(ctx, feed)
	if err != nil {
		notifier.completed(feed, 0, err)
		return nil, err
	}
	if skipped {
		s.logger.Info("feed not modified", "id", feed.ID, "title", feed.Title)
		notifier.skipped(feed, FetchSkipNotModified)
		return nil, nil
	}

	result, err := s.persistFetchedFeed(ctx, feed, document)
	if err != nil {
		notifier.completed(feed, 0, err)
		return nil, err
	}

	newEntries, count := s.resolveFetchedEntries(ctx, feed, result)
	notifier.completed(feed, count, nil)
	return newEntries, nil
}

// GetEntry retrieves a single entry by its primary key.
func (s *Service) GetEntry(ctx context.Context, id int64) (*Entry, error) {
	return s.store.GetEntry(ctx, id)
}

// ListEntries retrieves stored entries matching the given filter criteria.
// Results are ordered by fetched_at descending (newest first). It delegates
// directly to the store without additional business logic.
func (s *Service) ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error) {
	return s.store.ListEntries(ctx, filter)
}
