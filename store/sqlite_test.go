package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

func newTestStore(t *testing.T) *SQLiteStore {
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
	if !errors.Is(err, core.ErrFeedAlreadyExists) {
		t.Fatalf("expected ErrFeedAlreadyExists, got %v", err)
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
	if !errors.Is(err, core.ErrFeedNotFound) {
		t.Fatalf("expected ErrFeedNotFound, got %v", err)
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
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
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
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1", Link: "https://example.com/1", PublishedAt: &now, Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2", Link: "https://example.com/2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
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
			FeedID:       feed.ID,
			GUID:         "guid-full",
			Title:        "Full Entry",
			Link:         "https://example.com/full",
			Summary:      "Short summary",
			Content:      "<p>Full HTML content</p>",
			Author:       "John Doe",
			ImageURL:     "https://example.com/image.jpg",
			Categories:   json.RawMessage(`["go","rss","tech"]`),
			PublishedAt:  &now,
			UpdatedAt:    &updated,
			Enclosures:   json.RawMessage(`[{"url":"https://example.com/ep1.mp3","length":"12345","type":"audio/mpeg"}]`),
			Authors:      json.RawMessage(`[{"name":"John Doe","email":"john@example.com","uri":"https://john.example.com"}]`),
			Links:        json.RawMessage(`[{"href":"https://example.com/full","rel":"alternate","type":"text/html","hreflang":"","title":"","length":""}]`),
			Contributors: json.RawMessage(`[{"name":"Jane","email":"jane@example.com","uri":""}]`),
			Rights:       "Copyright 2026",
			Source:       json.RawMessage(`{"title":"Original","id":"urn:uuid:source","updated":"2026-01-01T00:00:00Z"}`),
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
	if string(e.Categories) != `["go","rss","tech"]` {
		t.Errorf("Categories = %q, want %q", e.Categories, `["go","rss","tech"]`)
	}
	if e.UpdatedAt == nil {
		t.Error("expected UpdatedAt to be set")
	}
	if string(e.Enclosures) != `[{"url":"https://example.com/ep1.mp3","length":"12345","type":"audio/mpeg"}]` {
		t.Errorf("Enclosures = %q, want JSON with podcast enclosure", e.Enclosures)
	}
	if string(e.Authors) != `[{"name":"John Doe","email":"john@example.com","uri":"https://john.example.com"}]` {
		t.Errorf("Authors = %q", e.Authors)
	}
	if string(e.Links) != `[{"href":"https://example.com/full","rel":"alternate","type":"text/html","hreflang":"","title":"","length":""}]` {
		t.Errorf("Links = %q", e.Links)
	}
	if string(e.Contributors) != `[{"name":"Jane","email":"jane@example.com","uri":""}]` {
		t.Errorf("Contributors = %q", e.Contributors)
	}
	if e.Rights != "Copyright 2026" {
		t.Errorf("Rights = %q, want %q", e.Rights, "Copyright 2026")
	}
	if string(e.Source) != `{"title":"Original","id":"urn:uuid:source","updated":"2026-01-01T00:00:00Z"}` {
		t.Errorf("Source = %q", e.Source)
	}
}

func TestAddEntriesDeduplication(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	entries := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	}
	_, _ = s.AddEntries(ctx, entries)

	// Same GUID should be skipped
	dupes := []*core.Entry{
		{FeedID: feed.ID, GUID: "guid-1", Title: "Entry 1 Updated", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "guid-2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
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
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "3", Title: "Entry 3", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
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
		{FeedID: feed1.ID, GUID: "a1", Title: "A1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed1.ID, GUID: "a2", Title: "A2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed2.ID, GUID: "b1", Title: "B1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
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

func TestUpdateFeed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Old Title"}
	_ = s.AddFeed(ctx, feed)

	newTitle := "New Title"
	err := s.UpdateFeed(ctx, feed.ID, core.FeedUpdate{Title: &newTitle})
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

func TestUpdateFeedCacheHeaders(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
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

func TestMarkEntryReadUnread(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	entryID := entries[0].ID

	// Mark as read.
	if err := s.MarkEntryRead(ctx, entryID); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	entries, _ = s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	if entries[0].ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}

	// Unread filter should exclude it.
	unread, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 0 {
		t.Errorf("expected 0 unread entries, got %d", len(unread))
	}

	// Mark as unread.
	if err := s.MarkEntryUnread(ctx, entryID); err != nil {
		t.Fatalf("MarkEntryUnread failed: %v", err)
	}

	unread, _ = s.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 1 {
		t.Errorf("expected 1 unread entry, got %d", len(unread))
	}
}

func TestMarkEntriesReadUnread(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := s.MarkEntriesRead(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesRead failed: %v", err)
	}

	unread, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 0 {
		t.Errorf("expected 0 unread entries, got %d", len(unread))
	}

	if err := s.MarkEntriesUnread(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesUnread failed: %v", err)
	}

	unread, _ = s.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 2 {
		t.Errorf("expected 2 unread entries, got %d", len(unread))
	}
}

func TestTags(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	// Add tags.
	if err := s.AddTag(ctx, feed1.ID, "tech"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if err := s.AddTag(ctx, feed1.ID, "go"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if err := s.AddTag(ctx, feed2.ID, "tech"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	// List tags for feed1.
	tags, err := s.ListTags(ctx, feed1.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("got %d tags, want 2", len(tags))
	}

	// List all tags.
	allTags, err := s.ListAllTags(ctx)
	if err != nil {
		t.Fatalf("ListAllTags failed: %v", err)
	}
	if len(allTags) != 2 {
		t.Errorf("got %d tags, want 2", len(allTags))
	}

	// List feeds by tag.
	techFeeds, err := s.ListFeedsByTag(ctx, "tech")
	if err != nil {
		t.Fatalf("ListFeedsByTag failed: %v", err)
	}
	if len(techFeeds) != 2 {
		t.Errorf("got %d feeds with 'tech' tag, want 2", len(techFeeds))
	}

	// Remove tag.
	if err := s.RemoveTag(ctx, feed1.ID, "go"); err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}
	tags, _ = s.ListTags(ctx, feed1.ID)
	if len(tags) != 1 {
		t.Errorf("got %d tags after remove, want 1", len(tags))
	}

	// Duplicate add should be idempotent.
	if err := s.AddTag(ctx, feed1.ID, "tech"); err != nil {
		t.Fatalf("duplicate AddTag failed: %v", err)
	}
}

func TestListFeedTags(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	_ = s.AddTag(ctx, feed1.ID, "tech")
	_ = s.AddTag(ctx, feed1.ID, "go")
	_ = s.AddTag(ctx, feed2.ID, "news")

	feedTags, err := s.ListFeedTags(ctx)
	if err != nil {
		t.Fatalf("ListFeedTags failed: %v", err)
	}

	if len(feedTags[feed1.ID]) != 2 {
		t.Fatalf("got %d tags for feed1, want 2", len(feedTags[feed1.ID]))
	}
	if feedTags[feed1.ID][0].Name != "go" || feedTags[feed1.ID][1].Name != "tech" {
		t.Fatalf("unexpected tag order for feed1: %+v", feedTags[feed1.ID])
	}
	if len(feedTags[feed2.ID]) != 1 || feedTags[feed2.ID][0].Name != "news" {
		t.Fatalf("unexpected tags for feed2: %+v", feedTags[feed2.ID])
	}
}

func TestListEntriesFilterByTag(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed1.ID, GUID: "a1", Title: "A1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed2.ID, GUID: "b1", Title: "B1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	_ = s.AddTag(ctx, feed1.ID, "tagged")

	result, err := s.ListEntries(ctx, core.EntryFilter{Tag: "tagged", Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries with tag filter failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d entries, want 1", len(result))
	}
	if len(result) > 0 && result[0].Title != "A1" {
		t.Errorf("Title = %q, want %q", result[0].Title, "A1")
	}
}

func TestSearchEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Golang Tutorial", Summary: "Learn Go", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Python Guide", Summary: "Learn Python", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "3", Title: "Rust Basics", Content: "Rust is great for golang interop", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	results, err := s.SearchEntries(ctx, "golang", 10)
	if err != nil {
		t.Fatalf("SearchEntries failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (title match + content match)", len(results))
	}

	results, err = s.SearchEntries(ctx, "python", 10)
	if err != nil {
		t.Fatalf("SearchEntries failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestStarUnstarEntry(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	id := entries[0].ID

	if err := s.StarEntry(ctx, id); err != nil {
		t.Fatalf("StarEntry failed: %v", err)
	}

	entries, _ = s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	if entries[0].StarredAt == nil {
		t.Error("expected StarredAt to be set")
	}

	// Starred filter.
	starred, _ := s.ListEntries(ctx, core.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 1 {
		t.Errorf("got %d starred, want 1", len(starred))
	}

	if err := s.UnstarEntry(ctx, id); err != nil {
		t.Fatalf("UnstarEntry failed: %v", err)
	}

	starred, _ = s.ListEntries(ctx, core.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 0 {
		t.Errorf("got %d starred after unstar, want 0", len(starred))
	}
}

func TestStarUnstarEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := s.StarEntries(ctx, ids); err != nil {
		t.Fatalf("StarEntries failed: %v", err)
	}

	starred, _ := s.ListEntries(ctx, core.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 2 {
		t.Errorf("got %d starred, want 2", len(starred))
	}

	if err := s.UnstarEntries(ctx, ids); err != nil {
		t.Fatalf("UnstarEntries failed: %v", err)
	}

	starred, _ = s.ListEntries(ctx, core.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 0 {
		t.Errorf("got %d starred after unstar, want 0", len(starred))
	}
}

func TestFeedHealthTracking(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	// Record errors.
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

	// Hit the threshold (5 total).
	_ = s.RecordFeedError(ctx, feed.ID, "timeout")
	_ = s.RecordFeedError(ctx, feed.ID, "timeout")

	got, _ = s.GetFeed(ctx, feed.ID)
	if !got.Disabled {
		t.Error("expected feed to be auto-disabled after 5 errors")
	}

	// Reset.
	_ = s.ResetFeedError(ctx, feed.ID)
	got, _ = s.GetFeed(ctx, feed.ID)
	if got.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d after reset, want 0", got.ErrorCount)
	}
}

func TestSetFeedDisabled(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
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

func TestFeedStats(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "E1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "E2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	_ = s.MarkEntryRead(ctx, entries[0].ID)
	_ = s.StarEntry(ctx, entries[0].ID)

	stats, err := s.FeedStats(ctx)
	if err != nil {
		t.Fatalf("FeedStats failed: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("got %d stats, want 1", len(stats))
	}
	if stats[0].TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", stats[0].TotalCount)
	}
	if stats[0].UnreadCount != 1 {
		t.Errorf("UnreadCount = %d, want 1", stats[0].UnreadCount)
	}
	if stats[0].StarredCount != 1 {
		t.Errorf("StarredCount = %d, want 1", stats[0].StarredCount)
	}
}

func TestCleanupEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Old Entry", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Starred Old", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	// Star one entry — should survive cleanup.
	_ = s.StarEntry(ctx, entries[0].ID)

	// Delete everything older than 0 (i.e., everything).
	deleted, err := s.CleanupEntries(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CleanupEntries failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1 (starred should survive)", deleted)
	}

	remaining, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if len(remaining) != 1 {
		t.Errorf("remaining = %d, want 1", len(remaining))
	}
}

func TestGetEntry(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Link: "https://example.com/1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	id := entries[0].ID

	got, err := s.GetEntry(ctx, id)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if got.Title != "Entry 1" {
		t.Errorf("Title = %q, want %q", got.Title, "Entry 1")
	}
}

func TestGetEntryNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetEntry(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent entry")
	}
	if !errors.Is(err, core.ErrEntryNotFound) {
		t.Fatalf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestFindDuplicateEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed1.ID, GUID: "a1", Title: "Shared Article", Link: "https://example.com/article", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed2.ID, GUID: "b1", Title: "Same Article", Link: "https://example.com/article", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed2.ID, GUID: "b2", Title: "Different Article", Link: "https://example.com/other", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	// Find the first entry (from feed1).
	var targetID int64
	for _, e := range entries {
		if e.GUID == "a1" {
			targetID = e.ID
			break
		}
	}

	dupes, err := s.FindDuplicateEntries(ctx, targetID)
	if err != nil {
		t.Fatalf("FindDuplicateEntries failed: %v", err)
	}
	if len(dupes) != 1 {
		t.Errorf("got %d duplicates, want 1", len(dupes))
	}
	if len(dupes) > 0 && dupes[0].Title != "Same Article" {
		t.Errorf("duplicate title = %q, want %q", dupes[0].Title, "Same Article")
	}
}
