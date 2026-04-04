package core

import (
	"context"
	"fmt"
	"log/slog"
)

// TagManager owns tag CRUD and feed-tag associations.
type TagManager struct {
	store  TagStore
	logger *slog.Logger
}

// NewTagManager creates a tag domain service.
func NewTagManager(store TagStore, logger *slog.Logger) *TagManager {
	return &TagManager{store: store, logger: normalizeLogger(logger)}
}

// AddTag associates a tag with a feed.
func (m *TagManager) AddTag(ctx context.Context, feedID int64, tagName string) error {
	if err := m.store.AddTag(ctx, feedID, tagName); err != nil {
		return fmt.Errorf("add tag: %w", err)
	}
	m.logger.Info("tag added", "feed_id", feedID, "tag", tagName)
	return nil
}

// RemoveTag removes a tag association from a feed.
func (m *TagManager) RemoveTag(ctx context.Context, feedID int64, tagName string) error {
	if err := m.store.RemoveTag(ctx, feedID, tagName); err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	m.logger.Info("tag removed", "feed_id", feedID, "tag", tagName)
	return nil
}

// ListTags returns all tags for a feed.
func (m *TagManager) ListTags(ctx context.Context, feedID int64) ([]Tag, error) {
	return m.store.ListTags(ctx, feedID)
}

// ListAllTags returns every tag in the system.
func (m *TagManager) ListAllTags(ctx context.Context) ([]Tag, error) {
	return m.store.ListAllTags(ctx)
}

// ListFeedTags returns all feed-tag associations keyed by feed ID.
func (m *TagManager) ListFeedTags(ctx context.Context) (map[int64][]Tag, error) {
	return m.store.ListFeedTags(ctx)
}

// ListFeedsByTag returns all feeds with the given tag.
func (m *TagManager) ListFeedsByTag(ctx context.Context, tagName string) ([]*Feed, error) {
	return m.store.ListFeedsByTag(ctx, tagName)
}

func (s *Service) AddTag(ctx context.Context, feedID int64, tagName string) error {
	return s.tags.AddTag(ctx, feedID, tagName)
}

func (s *Service) RemoveTag(ctx context.Context, feedID int64, tagName string) error {
	return s.tags.RemoveTag(ctx, feedID, tagName)
}

func (s *Service) ListTags(ctx context.Context, feedID int64) ([]Tag, error) {
	return s.tags.ListTags(ctx, feedID)
}

func (s *Service) ListAllTags(ctx context.Context) ([]Tag, error) {
	return s.tags.ListAllTags(ctx)
}

func (s *Service) ListFeedTags(ctx context.Context) (map[int64][]Tag, error) {
	return s.tags.ListFeedTags(ctx)
}

func (s *Service) ListFeedsByTag(ctx context.Context, tagName string) ([]*Feed, error) {
	return s.tags.ListFeedsByTag(ctx, tagName)
}
