package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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

	feedID := feed.ID
	result, err := s.ListEntries(ctx, core.EntryFilter{FeedID: &feedID, Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d entries, want 1", len(result))
	}

	e := result[0]
	want := entries[0]
	// Timestamps are stored as UTC in SQLite, so we exclude them from the diff
	// and check non-nil separately.
	if diff := cmp.Diff(want, e,
		cmpopts.IgnoreFields(core.Entry{}, "ID", "FetchedAt", "ReadAt", "StarredAt", "PublishedAt", "UpdatedAt"),
		cmpopts.IgnoreUnexported(core.Entry{}),
	); diff != "" {
		t.Errorf("entry mismatch (-want +got):\n%s", diff)
	}
	if e.PublishedAt == nil {
		t.Error("expected PublishedAt to be set")
	}
	if e.UpdatedAt == nil {
		t.Error("expected UpdatedAt to be set")
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

func TestCountEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*core.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, core.EntryFilter{Limit: 1})
	_ = s.MarkEntryRead(ctx, entries[0].ID)

	total, err := s.CountEntries(ctx, core.EntryFilter{})
	if err != nil {
		t.Fatalf("CountEntries failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}

	unread, err := s.CountEntries(ctx, core.EntryFilter{UnreadOnly: true})
	if err != nil {
		t.Fatalf("CountEntries unread failed: %v", err)
	}
	if unread != 1 {
		t.Errorf("unread = %d, want 1", unread)
	}
}

func TestListEntriesAndCountEntriesCombinedFilters(t *testing.T) {
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

	_ = s.AddTag(ctx, feed1.ID, "tech")
	_ = s.AddTag(ctx, feed2.ID, "news")

	allEntries, err := s.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(allEntries) != 3 {
		t.Fatalf("got %d entries, want 3", len(allEntries))
	}

	var feed1ReadID int64
	for _, entry := range allEntries {
		if entry.FeedID == feed1.ID {
			feed1ReadID = entry.ID
			break
		}
	}
	if feed1ReadID == 0 {
		t.Fatal("expected a feed1 entry ID")
	}

	if err := s.MarkEntryRead(ctx, feed1ReadID); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}
	if err := s.StarEntry(ctx, feed1ReadID); err != nil {
		t.Fatalf("StarEntry failed: %v", err)
	}

	feed1ID := feed1.ID
	feed2ID := feed2.ID
	testCases := []struct {
		name   string
		filter core.EntryFilter
		want   int
	}{
		{name: "all", filter: core.EntryFilter{}, want: 3},
		{name: "feed1", filter: core.EntryFilter{FeedID: &feed1ID}, want: 2},
		{name: "tag", filter: core.EntryFilter{Tag: "tech"}, want: 2},
		{name: "unread", filter: core.EntryFilter{UnreadOnly: true}, want: 2},
		{name: "starred", filter: core.EntryFilter{StarredOnly: true}, want: 1},
		{name: "feed1 unread tag", filter: core.EntryFilter{FeedID: &feed1ID, UnreadOnly: true, Tag: "tech"}, want: 1},
		{name: "feed1 starred tag", filter: core.EntryFilter{FeedID: &feed1ID, StarredOnly: true, Tag: "tech"}, want: 1},
		{name: "feed2 tech", filter: core.EntryFilter{FeedID: &feed2ID, Tag: "tech"}, want: 0},
	}

	for _, tc := range testCases {
		entries, err := s.ListEntries(ctx, tc.filter)
		if err != nil {
			t.Fatalf("%s: ListEntries failed: %v", tc.name, err)
		}
		count, err := s.CountEntries(ctx, tc.filter)
		if err != nil {
			t.Fatalf("%s: CountEntries failed: %v", tc.name, err)
		}
		if len(entries) != tc.want {
			t.Fatalf("%s: got %d entries, want %d", tc.name, len(entries), tc.want)
		}
		if count != tc.want {
			t.Fatalf("%s: count = %d, want %d", tc.name, count, tc.want)
		}
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
	_ = s.StarEntry(ctx, entries[0].ID)

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

func TestRunInTxCommitsOnSuccess(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Before"}
	_ = s.AddFeed(ctx, feed)

	err := s.RunInTx(ctx, func(ctx context.Context) error {
		return s.SetFeedDisabled(ctx, feed.ID, true)
	})
	if err != nil {
		t.Fatalf("RunInTx failed: %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if !got.Disabled {
		t.Error("expected feed to be disabled after committed transaction")
	}
}

func TestRunInTxRollsBackOnError(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Before"}
	_ = s.AddFeed(ctx, feed)

	boom := errors.New("intentional failure")
	err := s.RunInTx(ctx, func(ctx context.Context) error {
		// Change disabled state inside the transaction, then fail.
		_ = s.SetFeedDisabled(ctx, feed.ID, true)
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.Disabled {
		t.Error("expected feed to remain enabled after rolled-back transaction")
	}
}

func TestRunInTxReusesOuterTransaction(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Before"}
	_ = s.AddFeed(ctx, feed)

	boom := errors.New("outer failure")
	err := s.RunInTx(ctx, func(ctx context.Context) error {
		// Inner RunInTx reuses the outer transaction.
		_ = s.RunInTx(ctx, func(ctx context.Context) error {
			return s.SetFeedDisabled(ctx, feed.ID, true)
		})
		return boom // outer error causes rollback of inner change too
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if got.Disabled {
		t.Error("expected feed to remain enabled: inner change rolled back with outer tx")
	}
}

func TestEnableFeedRollsBackOnPartialFailure(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Add a feed with an error so it gets disabled.
	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Feed"}
	_ = s.AddFeed(ctx, feed)
	_ = s.SetFeedDisabled(ctx, feed.ID, true)
	_ = s.RecordFeedError(ctx, feed.ID, "some error")

	// Simulate a transaction where SetFeedDisabled succeeds but ResetFeedError
	// would fail by wrapping both in a RunInTx that we force to roll back.
	boom := errors.New("reset error failure")
	err := s.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.SetFeedDisabled(ctx, feed.ID, false); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}

	got, _ := s.GetFeed(ctx, feed.ID)
	if !got.Disabled {
		t.Error("expected feed to remain disabled after rolled-back transaction")
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
