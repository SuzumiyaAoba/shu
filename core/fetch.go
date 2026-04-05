package core

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type fetcherStore interface {
	FeedStore
	FeedHealthStore
	EntryStore
	TxRunner
}

// Fetcher owns feed download, parse, and persistence workflows.
type Fetcher struct {
	store      fetcherStore
	logger     *slog.Logger
	downloader feedDownloader
	persister  feedPersister
}

// NewFetcher creates a fetch domain service.
func NewFetcher(store fetcherStore, logger *slog.Logger, client *http.Client) *Fetcher {
	logger = normalizeLogger(logger)
	return &Fetcher{
		store:      store,
		logger:     logger,
		downloader: newHTTPFeedDownloader(store, logger, client),
		persister:  newStoreFeedPersister(store, logger),
	}
}

func (f *Fetcher) setHTTPClient(client *http.Client) {
	if downloader, ok := f.downloader.(*httpFeedDownloader); ok {
		downloader.setHTTPClient(client)
	}
}

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID, then
// stores any new entries that are not already in the database.
//
// For Atom feeds, the response body is parsed twice: once with gofeed's
// universal parser for common fields, and once with the Atom-specific parser
// to capture fields lost in translation (Contributors, Rights, Source,
// Author URIs, structured Categories, and full Link metadata).
//
// It returns a slice containing only the newly inserted entries.
func (f *Fetcher) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) {
	return f.FetchFeedWithObserver(ctx, feedID, nil)
}

// FetchFeedWithObserver downloads and parses a single feed while emitting
// structured progress events to observer.
func (f *Fetcher) FetchFeedWithObserver(ctx context.Context, feedID int64, observer FetchObserver) ([]*Entry, error) {
	notifier := newFetchNotifier(observer)
	return f.fetchFeedByID(ctx, feedID, notifier)
}

func (f *Fetcher) fetchFeedByID(ctx context.Context, feedID int64, notifier *fetchNotifier) ([]*Entry, error) {
	feed, err := f.store.GetFeed(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("get feed %d: %w", feedID, err)
	}

	return f.fetchFeed(ctx, feed, notifier)
}

func (f *Fetcher) fetchFeed(ctx context.Context, feed *Feed, notifier *fetchNotifier) ([]*Entry, error) {
	notifier.started(feed)

	if feed.Disabled {
		f.logger.Warn("feed disabled, skipping", "id", feed.ID, "title", feed.Title)
		notifier.skipped(feed, FetchSkipDisabled)
		return nil, nil
	}

	document, skipped, err := f.downloader.download(ctx, feed)
	if err != nil {
		notifier.completed(feed, 0, err)
		return nil, err
	}
	if skipped {
		f.logger.Info("feed not modified", "id", feed.ID, "title", feed.Title)
		notifier.skipped(feed, FetchSkipNotModified)
		return nil, nil
	}

	result, err := f.persister.persist(ctx, feed, document)
	if err != nil {
		notifier.completed(feed, 0, err)
		return nil, err
	}

	newEntries, count := f.resolveFetchedEntries(ctx, feed, result)
	notifier.completed(feed, count, nil)
	return newEntries, nil
}

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID.
func (s *Service) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) {
	return s.fetcher.FetchFeed(ctx, feedID)
}

func (s *Service) FetchFeedWithObserver(ctx context.Context, feedID int64, observer FetchObserver) ([]*Entry, error) {
	return s.fetcher.FetchFeedWithObserver(ctx, feedID, observer)
}
