package store

import (
	"context"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

func newTestStore(t *testing.T) Store {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAddFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{
		URL:     "https://example.com/feed.xml",
		Title:   "Example Feed",
		SiteURL: "https://example.com",
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

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Feed 1"}
	if err := s.AddFeed(ctx, feed); err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	dup := &core.Feed{URL: "https://example.com/feed.xml", Title: "Feed 2"}
	err := s.AddFeed(ctx, dup)
	if err == nil {
		t.Error("expected error for duplicate URL")
	}
}

func TestGetFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{
		URL:     "https://example.com/feed.xml",
		Title:   "Example Feed",
		SiteURL: "https://example.com",
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
}

func TestGetFeedNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetFeed(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}
}

func TestListFeeds(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_ = s.AddFeed(ctx, &core.Feed{URL: "https://a.com/feed", Title: "A"})
	_ = s.AddFeed(ctx, &core.Feed{URL: "https://b.com/feed", Title: "B"})

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

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
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

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1"},
	}
	_, _ = s.AddEntries(ctx, entries)

	_ = s.RemoveFeed(ctx, feed.ID)

	feedID := feed.ID
	result, err := s.ListEntries(ctx, core.EntryFilter{FeedID: &feedID, Limit: 10})
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

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
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

func TestAddEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	now := time.Now()
	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1", Link: "https://example.com/1", PublishedAt: &now},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2", Link: "https://example.com/2"},
	}

	inserted, err := s.AddEntries(ctx, entries)
	if err != nil {
		t.Fatalf("AddEntries failed: %v", err)
	}
	if inserted != 2 {
		t.Errorf("inserted = %d, want 2", inserted)
	}
}

func TestAddEntriesDeduplication(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1"},
	}
	_, _ = s.AddEntries(ctx, entries)

	// Same GUID should be skipped
	dupes := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1 Updated"},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2"},
	}
	inserted, err := s.AddEntries(ctx, dupes)
	if err != nil {
		t.Fatalf("AddEntries failed: %v", err)
	}
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1 (guid-1 should be skipped)", inserted)
	}
}

func TestAddEntriesEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	inserted, err := s.AddEntries(ctx, nil)
	if err != nil {
		t.Fatalf("AddEntries with nil failed: %v", err)
	}
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0", inserted)
	}
}

func TestListEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1"},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2"},
		{FeedID: feed.ID, GUID: "3", Title: "Entry 3"},
	}
	_, _ = s.AddEntries(ctx, entries)

	result, err := s.ListEntries(ctx, core.EntryFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d entries, want 2", len(result))
	}
}

func TestListEntriesFilterByFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed1.ID, GUID: "a1", Title: "A1"},
		{FeedID: feed1.ID, GUID: "a2", Title: "A2"},
		{FeedID: feed2.ID, GUID: "b1", Title: "B1"},
	})

	feedID := feed1.ID
	result, err := s.ListEntries(ctx, core.EntryFilter{FeedID: &feedID, Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d entries, want 2", len(result))
	}
}
