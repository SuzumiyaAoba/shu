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
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
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
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1", Link: "https://example.com/1", PublishedAt: &now, Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2", Link: "https://example.com/2", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
	}

	inserted, err := s.AddEntries(ctx, entries)
	if err != nil {
		t.Fatalf("AddEntries failed: %v", err)
	}
	if inserted != 2 {
		t.Errorf("inserted = %d, want 2", inserted)
	}
}

func TestAddEntriesExpandedFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	now := time.Now()
	updated := now.Add(time.Hour)
	entries := []*core.Entry{
		{
			FeedID:     feed.ID,
			GUID:       "guid-full",
			Title:      "Full Entry",
			Link:       "https://example.com/full",
			Summary:    "Short summary",
			Content:    "<p>Full HTML content</p>",
			Author:     "John Doe",
			ImageURL:   "https://example.com/image.jpg",
			Categories: `["go","rss","tech"]`,
			PublishedAt: &now,
			UpdatedAt:  &updated,
			Enclosures:   `[{"url":"https://example.com/ep1.mp3","length":"12345","type":"audio/mpeg"}]`,
			Authors:      `[{"name":"John Doe","email":"john@example.com","uri":"https://john.example.com"}]`,
			Links:        `[{"href":"https://example.com/full","rel":"alternate","type":"text/html","hreflang":"","title":"","length":""}]`,
			Contributors: `[{"name":"Jane","email":"jane@example.com","uri":""}]`,
			Rights:       "Copyright 2026",
			Source:        `{"title":"Original","id":"urn:uuid:source","updated":"2026-01-01T00:00:00Z"}`,
		},
	}

	inserted, err := s.AddEntries(ctx, entries)
	if err != nil {
		t.Fatalf("AddEntries failed: %v", err)
	}
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1", inserted)
	}

	// Verify round-trip via ListEntries.
	feedID := feed.ID
	result, err := s.ListEntries(ctx, core.EntryFilter{FeedID: &feedID, Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d entries, want 1", len(result))
	}

	e := result[0]
	if e.Content != "<p>Full HTML content</p>" {
		t.Errorf("Content = %q, want %q", e.Content, "<p>Full HTML content</p>")
	}
	if e.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", e.Author, "John Doe")
	}
	if e.ImageURL != "https://example.com/image.jpg" {
		t.Errorf("ImageURL = %q, want %q", e.ImageURL, "https://example.com/image.jpg")
	}
	if e.Categories != `["go","rss","tech"]` {
		t.Errorf("Categories = %q, want %q", e.Categories, `["go","rss","tech"]`)
	}
	if e.UpdatedAt == nil {
		t.Error("expected UpdatedAt to be set")
	}
	if e.Enclosures != `[{"url":"https://example.com/ep1.mp3","length":"12345","type":"audio/mpeg"}]` {
		t.Errorf("Enclosures = %q, want JSON with podcast enclosure", e.Enclosures)
	}
	if e.Authors != `[{"name":"John Doe","email":"john@example.com","uri":"https://john.example.com"}]` {
		t.Errorf("Authors = %q", e.Authors)
	}
	if e.Links != `[{"href":"https://example.com/full","rel":"alternate","type":"text/html","hreflang":"","title":"","length":""}]` {
		t.Errorf("Links = %q", e.Links)
	}
	if e.Contributors != `[{"name":"Jane","email":"jane@example.com","uri":""}]` {
		t.Errorf("Contributors = %q", e.Contributors)
	}
	if e.Rights != "Copyright 2026" {
		t.Errorf("Rights = %q, want %q", e.Rights, "Copyright 2026")
	}
	if e.Source != `{"title":"Original","id":"urn:uuid:source","updated":"2026-01-01T00:00:00Z"}` {
		t.Errorf("Source = %q", e.Source)
	}
}

func TestAddEntriesDeduplication(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
	}
	_, _ = s.AddEntries(ctx, entries)

	// Same GUID should be skipped
	dupes := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1 Updated", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
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
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed.ID, GUID: "3", Title: "Entry 3", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
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
		{FeedID: feed1.ID, GUID: "a1", Title: "A1", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed1.ID, GUID: "a2", Title: "A2", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
		{FeedID: feed2.ID, GUID: "b1", Title: "B1", Categories: "[]", Enclosures: "[]", Authors: "[]", Links: "[]", Contributors: "[]"},
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
