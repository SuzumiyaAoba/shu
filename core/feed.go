package core

import (
	"context"
	"fmt"

	"github.com/mmcdole/gofeed"
)

func (s *Service) AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error) {
	fp := gofeed.NewParser()
	fp.Client = s.client

	parsed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %s: %w", url, err)
	}

	title := parsed.Title
	if titleOverride != "" {
		title = titleOverride
	}

	siteURL := ""
	if parsed.Link != "" {
		siteURL = parsed.Link
	}

	feed := &Feed{
		URL:     url,
		Title:   title,
		SiteURL: siteURL,
	}

	if err := s.store.AddFeed(ctx, feed); err != nil {
		return nil, fmt.Errorf("store feed: %w", err)
	}

	s.logger.Info("feed added", "id", feed.ID, "title", feed.Title, "url", feed.URL)
	return feed, nil
}

func (s *Service) ListFeeds(ctx context.Context) ([]*Feed, error) {
	return s.store.ListFeeds(ctx)
}

func (s *Service) RemoveFeed(ctx context.Context, id int64) error {
	if err := s.store.RemoveFeed(ctx, id); err != nil {
		return fmt.Errorf("remove feed %d: %w", id, err)
	}
	s.logger.Info("feed removed", "id", id)
	return nil
}
