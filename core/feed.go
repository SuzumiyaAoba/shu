package core

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mmcdole/gofeed"
)

type feedManagerStore interface {
	FeedStore
	FeedHealthStore
	TxRunner
}

// FeedManager owns feed CRUD and enabled/disabled state changes.
type FeedManager struct {
	store  feedManagerStore
	logger *slog.Logger
	client *http.Client
}

// NewFeedManager creates a feed domain service.
func NewFeedManager(store feedManagerStore, logger *slog.Logger, client *http.Client) *FeedManager {
	return &FeedManager{
		store:  store,
		logger: normalizeLogger(logger),
		client: normalizeHTTPClient(client),
	}
}

func (m *FeedManager) setHTTPClient(client *http.Client) {
	m.client = normalizeHTTPClient(client)
}

// AddFeed registers a new RSS/Atom feed by its URL.
//
// The method performs the following steps:
//  1. Fetches and parses the feed document at the given URL to validate it and
//     extract metadata (title, site URL, description, language, image, type).
//  2. If titleOverride is non-empty, it is used instead of the title found in
//     the feed document.
//  3. Persists the feed record via the store. On success the returned [Feed]
//     has its ID and AddedAt fields populated.
//
// An error is returned if the URL is unreachable, the document is not a valid
// feed, or the store rejects the insertion (e.g. duplicate URL).
func (m *FeedManager) AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error) {
	body, err := fetchBody(ctx, m.client, url)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %s: %w", url, err)
	}

	fp := gofeed.NewParser()
	parsed, err := fp.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: parse feed %s: %v", ErrInvalidFeed, url, err)
	}

	title := parsed.Title
	if titleOverride != "" {
		title = titleOverride
	}

	siteURL := ""
	if parsed.Link != "" {
		siteURL = parsed.Link
	}

	imageURL := ""
	if parsed.Image != nil {
		imageURL = parsed.Image.URL
	}

	feed := &Feed{
		URL:         url,
		Title:       title,
		SiteURL:     siteURL,
		Description: parsed.Description,
		Language:    parsed.Language,
		ImageURL:    imageURL,
		FeedType:    parsed.FeedType,
	}

	if err := m.store.AddFeed(ctx, feed); err != nil {
		return nil, fmt.Errorf("store feed: %w", err)
	}

	m.logger.With("feed_id", feed.ID, "feed_url", feed.URL).Info("feed added", "title", feed.Title)
	return feed, nil
}

// ListFeeds returns all registered feeds ordered by ID.
// It delegates directly to the store without additional business logic.
func (m *FeedManager) ListFeeds(ctx context.Context) ([]*Feed, error) {
	return m.store.ListFeeds(ctx)
}

// GetFeed retrieves a single feed by its primary key.
func (m *FeedManager) GetFeed(ctx context.Context, id int64) (*Feed, error) {
	return m.store.GetFeed(ctx, id)
}

// RemoveFeed deletes a feed and all of its associated entries (via cascade
// delete in the database). The id parameter is the feed's primary key.
func (m *FeedManager) RemoveFeed(ctx context.Context, id int64) error {
	if err := m.store.RemoveFeed(ctx, id); err != nil {
		return fmt.Errorf("remove feed %d: %w", id, err)
	}
	m.logger.With("feed_id", id).Info("feed removed")
	return nil
}

// AddFeedDirect registers a feed without performing any HTTP fetch or
// validation. The feed's URL must be non-empty. This is intended for bulk
// imports (e.g. OPML) where metadata comes from the import source rather
// than the feed server. Invalid URLs will fail when the feed is fetched.
func (m *FeedManager) AddFeedDirect(ctx context.Context, feed *Feed) error {
	if feed.URL == "" {
		return fmt.Errorf("feed URL is required")
	}
	if err := m.store.AddFeed(ctx, feed); err != nil {
		return fmt.Errorf("store feed: %w", err)
	}
	m.logger.With("feed_id", feed.ID, "feed_url", feed.URL).Info("feed added (direct)")
	return nil
}

func (s *Service) AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error) {
	return s.feeds.AddFeed(ctx, url, titleOverride)
}

func (s *Service) AddFeedDirect(ctx context.Context, feed *Feed) error {
	return s.feeds.AddFeedDirect(ctx, feed)
}

func (s *Service) ListFeeds(ctx context.Context) ([]*Feed, error) {
	return s.feeds.ListFeeds(ctx)
}

func (s *Service) GetFeed(ctx context.Context, id int64) (*Feed, error) {
	return s.feeds.GetFeed(ctx, id)
}

func (s *Service) RemoveFeed(ctx context.Context, id int64) error {
	return s.feeds.RemoveFeed(ctx, id)
}
