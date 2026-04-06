package fetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/SuzumiyaAoba/shu/model"
)

// Store defines the persistence operations needed by the fetch pipeline.
type Store interface {
	GetFeed(ctx context.Context, id int64) (*model.Feed, error)
	ListFeeds(ctx context.Context) ([]*model.Feed, error)
	AddEntries(ctx context.Context, entries []*model.Entry) (int, error)
	ListEntries(ctx context.Context, filter model.EntryFilter) ([]*model.Entry, error)
	UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
	RecordFeedError(ctx context.Context, id int64, errMsg string) error
	ResetFeedError(ctx context.Context, id int64) error
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Fetcher owns feed download, parse, and persistence workflows.
type Fetcher struct {
	store      Store
	logger     *slog.Logger
	downloader feedDownloader
	persister  feedPersister
}

// NewFetcher creates a fetch domain service.
func NewFetcher(store Store, logger *slog.Logger, client *http.Client) *Fetcher {
	logger = ensureLogger(logger)
	return &Fetcher{
		store:      store,
		logger:     logger,
		downloader: newHTTPFeedDownloader(store, logger, client),
		persister:  newStoreFeedPersister(store, logger),
	}
}

// SetHTTPClient replaces the HTTP client used for feed downloads.
func (f *Fetcher) SetHTTPClient(client *http.Client) {
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
func (f *Fetcher) FetchFeed(ctx context.Context, feedID int64) ([]*model.Entry, error) {
	return f.FetchFeedWithObserver(ctx, feedID, nil)
}

// FetchFeedWithObserver downloads and parses a single feed while emitting
// structured progress events to observer.
func (f *Fetcher) FetchFeedWithObserver(ctx context.Context, feedID int64, observer Observer) ([]*model.Entry, error) {
	n := newNotifier(observer)
	return f.fetchFeedByID(ctx, feedID, n)
}

func (f *Fetcher) fetchFeedByID(ctx context.Context, feedID int64, n *notifier) ([]*model.Entry, error) {
	feed, err := f.store.GetFeed(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("get feed %d: %w", feedID, err)
	}

	return f.fetchFeed(ctx, feed, n)
}

func (f *Fetcher) fetchFeed(ctx context.Context, feed *model.Feed, n *notifier) ([]*model.Entry, error) {
	n.started(feed.ID, feed.Title, feed.URL)

	if feed.Disabled {
		f.logger.Warn("feed disabled, skipping", "id", feed.ID, "title", feed.Title)
		n.skipped(feed.ID, feed.Title, feed.URL, SkipDisabled)
		return nil, nil
	}

	document, skipped, err := f.downloader.download(ctx, feed)
	if err != nil {
		n.completed(feed.ID, feed.Title, feed.URL, 0, err)
		return nil, err
	}
	if skipped {
		f.logger.Info("feed not modified", "id", feed.ID, "title", feed.Title)
		n.skipped(feed.ID, feed.Title, feed.URL, SkipNotModified)
		return nil, nil
	}

	result, err := f.persister.persist(ctx, feed, document)
	if err != nil {
		n.completed(feed.ID, feed.Title, feed.URL, 0, err)
		return nil, err
	}

	newEntries, count := f.resolveFetchedEntries(ctx, feed, result)
	n.completed(feed.ID, feed.Title, feed.URL, count, nil)
	return newEntries, nil
}

func (f *Fetcher) resolveFetchedEntries(ctx context.Context, feed *model.Feed, result *persistedEntries) ([]*model.Entry, int) {
	if result.inserted == 0 {
		return nil, 0
	}
	if result.inserted == len(result.entries) {
		return result.entries, len(result.entries)
	}

	feedID := feed.ID
	newEntries, err := f.store.ListEntries(ctx, model.EntryFilter{FeedID: &feedID, Limit: result.inserted})
	if err != nil {
		f.logger.Warn("failed to retrieve newly inserted entries", "id", feed.ID, "error", err)
		return result.entries[:result.inserted], result.inserted
	}
	return newEntries, len(newEntries)
}

func ensureLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func ensureHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{}
}
