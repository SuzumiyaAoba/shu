package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/model"
)

func TestAddFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{
		URL:         "https://example.com/feed.xml",
		Title:       "Example Feed",
		SiteURL:     "https://example.com",
		Description: "A test feed",
		Language:    "en",
		ImageURL:    "https://example.com/logo.png",
		FeedType:    "rss",
	}

	err := s.AddFeed(ctx, feed)
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	if feed.ID == 0 {
		t.Error("expected feed ID to be set")
	}
	if feed.AddedAt.IsZero() {
		t.Error("expected AddedAt to be set")
	}
}

func TestAddFeedDuplicateURL(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Feed 1"}
	if err := s.AddFeed(ctx, feed); err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	dup := &model.Feed{URL: "https://example.com/feed.xml", Title: "Feed 2"}
	err := s.AddFeed(ctx, dup)
	if err == nil {
		t.Error("expected error for duplicate URL")
	}
	if !errors.Is(err, model.ErrFeedAlreadyExists) {
		t.Fatalf("expected ErrFeedAlreadyExists, got %v", err)
	}
}

func TestGetFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{
		URL:         "https://example.com/feed.xml",
		Title:       "Example Feed",
		SiteURL:     "https://example.com",
		Description: "A test feed",
		Language:    "en",
		ImageURL:    "https://example.com/logo.png",
		FeedType:    "rss",
	}
	_ = s.AddFeed(ctx, feed)

	got, err := s.GetFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.URL != feed.URL {
		t.Errorf("URL = %q, want %q", got.URL, feed.URL)
	}
	if got.Title != feed.Title {
		t.Errorf("Title = %q, want %q", got.Title, feed.Title)
	}
	if got.Description != "A test feed" {
		t.Errorf("Description = %q, want %q", got.Description, "A test feed")
	}
	if got.Language != "en" {
		t.Errorf("Language = %q, want %q", got.Language, "en")
	}
	if got.ImageURL != "https://example.com/logo.png" {
		t.Errorf("ImageURL = %q, want %q", got.ImageURL, "https://example.com/logo.png")
	}
	if got.FeedType != "rss" {
		t.Errorf("FeedType = %q, want %q", got.FeedType, "rss")
	}
}

func TestGetFeedNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetFeed(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}
	if !errors.Is(err, model.ErrFeedNotFound) {
		t.Fatalf("expected ErrFeedNotFound, got %v", err)
	}
}

func TestListFeeds(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_ = s.AddFeed(ctx, &model.Feed{URL: "https://a.com/feed", Title: "A"})
	_ = s.AddFeed(ctx, &model.Feed{URL: "https://b.com/feed", Title: "B"})

	feeds, err := s.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("got %d feeds, want 2", len(feeds))
	}
}

func TestRemoveFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	err := s.RemoveFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("RemoveFeed failed: %v", err)
	}

	_, err = s.GetFeed(ctx, feed.ID)
	if err == nil {
		t.Error("expected error after removing feed")
	}
}

func TestRemoveFeedCascadesEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*model.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	}
	_, _ = s.AddEntries(ctx, entries)

	_ = s.RemoveFeed(ctx, feed.ID)

	feedID := feed.ID
	result, err := s.ListEntries(ctx, model.EntryFilter{FeedID: &feedID, Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after cascade delete, got %d", len(result))
	}
}

func TestUpdateFeedFetchedAt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	err := s.UpdateFeedFetchedAt(ctx, feed.ID)
	if err != nil {
		t.Fatalf("UpdateFeedFetchedAt failed: %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.FetchedAt == nil {
		t.Error("expected FetchedAt to be set")
	}
}

func TestSQLiteBusyTimeout(t *testing.T) {
	s := newTestStore(t)

	var timeout int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("query busy_timeout failed: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
}

func TestNewSQLiteStoreWithOptions(t *testing.T) {
	s, err := NewSQLiteStoreWithOptions(":memory:", &SQLiteOptions{
		MaxOpenConns: 2,
		BusyTimeout:  2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStoreWithOptions failed: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if got := s.db.Stats().MaxOpenConnections; got != 2 {
		t.Fatalf("MaxOpenConnections = %d, want 2", got)
	}

	var timeout int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("query busy_timeout failed: %v", err)
	}
	if timeout != 2000 {
		t.Fatalf("busy_timeout = %d, want 2000", timeout)
	}
}

func TestUpdateFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Old Title"}
	_ = s.AddFeed(ctx, feed)

	newTitle := "New Title"
	err := s.UpdateFeed(ctx, feed.ID, model.FeedUpdate{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.Title != "New Title" {
		t.Errorf("Title = %q, want %q", got.Title, "New Title")
	}
	if got.URL != "https://example.com/feed.xml" {
		t.Errorf("URL should be unchanged: %q", got.URL)
	}
}

func TestUpdateFeedFetchIntervalSec(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	interval := 3600
	err := s.UpdateFeed(ctx, feed.ID, model.FeedUpdate{FetchIntervalSec: &interval})
	if err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.FetchIntervalSec != 3600 {
		t.Errorf("FetchIntervalSec = %d, want 3600", got.FetchIntervalSec)
	}

	// Reset to 0
	zero := 0
	_ = s.UpdateFeed(ctx, feed.ID, model.FeedUpdate{FetchIntervalSec: &zero})
	got, _ = s.GetFeed(ctx, feed.ID)
	if got.FetchIntervalSec != 0 {
		t.Errorf("FetchIntervalSec = %d, want 0", got.FetchIntervalSec)
	}
}

func TestUpdateFeedCacheHeaders(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	err := s.UpdateFeedCacheHeaders(ctx, feed.ID, `"abc123"`, "Mon, 01 Jan 2026 00:00:00 GMT")
	if err != nil {
		t.Fatalf("UpdateFeedCacheHeaders failed: %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.ETag != `"abc123"` {
		t.Errorf("ETag = %q, want %q", got.ETag, `"abc123"`)
	}
	if got.LastModified != "Mon, 01 Jan 2026 00:00:00 GMT" {
		t.Errorf("LastModified = %q", got.LastModified)
	}
}

func TestFeedHealthTracking(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	for i := 0; i < 3; i++ {
		_ = s.RecordFeedError(ctx, feed.ID, "connection timeout")
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.ErrorCount != 3 {
		t.Errorf("ErrorCount = %d, want 3", got.ErrorCount)
	}
	if got.LastError != "connection timeout" {
		t.Errorf("LastError = %q", got.LastError)
	}
	if got.Disabled {
		t.Error("should not be disabled yet (threshold is 5)")
	}

	_ = s.RecordFeedError(ctx, feed.ID, "timeout")
	_ = s.RecordFeedError(ctx, feed.ID, "timeout")

	got, _ = s.GetFeed(ctx, feed.ID)
	if !got.Disabled {
		t.Error("expected feed to be auto-disabled after 5 errors")
	}

	_ = s.ResetFeedError(ctx, feed.ID)
	got, _ = s.GetFeed(ctx, feed.ID)
	if got.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d after reset, want 0", got.ErrorCount)
	}
}

func TestSetFeedDisabled(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_ = s.SetFeedDisabled(ctx, feed.ID, true)
	got, _ := s.GetFeed(ctx, feed.ID)
	if !got.Disabled {
		t.Error("expected disabled")
	}

	_ = s.SetFeedDisabled(ctx, feed.ID, false)
	got, _ = s.GetFeed(ctx, feed.ID)
	if got.Disabled {
		t.Error("expected enabled")
	}
}
