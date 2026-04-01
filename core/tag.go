package core

import (
	"context"
	"fmt"
)

// AddTag associates a tag with a feed.
func (s *Service) AddTag(ctx context.Context, feedID int64, tagName string) error {
	if err := s.store.AddTag(ctx, feedID, tagName); err != nil {
		return fmt.Errorf("add tag: %w", err)
	}
	s.logger.Info("tag added", "feed_id", feedID, "tag", tagName)
	return nil
}

// RemoveTag removes a tag association from a feed.
func (s *Service) RemoveTag(ctx context.Context, feedID int64, tagName string) error {
	if err := s.store.RemoveTag(ctx, feedID, tagName); err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	s.logger.Info("tag removed", "feed_id", feedID, "tag", tagName)
	return nil
}

// ListTags returns all tags for a feed.
func (s *Service) ListTags(ctx context.Context, feedID int64) ([]Tag, error) {
	return s.store.ListTags(ctx, feedID)
}

// ListAllTags returns every tag in the system.
func (s *Service) ListAllTags(ctx context.Context) ([]Tag, error) {
	return s.store.ListAllTags(ctx)
}

// ListFeedTags returns all feed-tag associations keyed by feed ID.
func (s *Service) ListFeedTags(ctx context.Context) (map[int64][]Tag, error) {
	return s.store.ListFeedTags(ctx)
}

// ListFeedsByTag returns all feeds with the given tag.
func (s *Service) ListFeedsByTag(ctx context.Context, tagName string) ([]*Feed, error) {
	return s.store.ListFeedsByTag(ctx, tagName)
}
