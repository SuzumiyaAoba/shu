package core

import (
	"bytes"
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"
)

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
func (s *Service) AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error) {
	body, err := s.fetchBody(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %s: %w", url, err)
	}

	fp := gofeed.NewParser()
	parsed, err := fp.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse feed %s: %w", url, err)
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

	if err := s.store.AddFeed(ctx, feed); err != nil {
		return nil, fmt.Errorf("store feed: %w", err)
	}

	s.logger.Info("feed added", "id", feed.ID, "title", feed.Title, "url", feed.URL)
	return feed, nil
}

// ListFeeds returns all registered feeds ordered by ID.
// It delegates directly to the store without additional business logic.
func (s *Service) ListFeeds(ctx context.Context) ([]*Feed, error) {
	return s.store.ListFeeds(ctx)
}

// RemoveFeed deletes a feed and all of its associated entries (via cascade
// delete in the database). The id parameter is the feed's primary key.
func (s *Service) RemoveFeed(ctx context.Context, id int64) error {
	if err := s.store.RemoveFeed(ctx, id); err != nil {
		return fmt.Errorf("remove feed %d: %w", id, err)
	}
	s.logger.Info("feed removed", "id", id)
	return nil
}
