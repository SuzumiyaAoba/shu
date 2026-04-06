package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SuzumiyaAoba/shu/model"
)

func TestMarkEntryReadUnread(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*model.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 1})
	entryID := entries[0].ID

	if err := s.MarkEntryRead(ctx, entryID); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	entries, _ = s.ListEntries(ctx, model.EntryFilter{Limit: 1})
	if entries[0].ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}

	unread, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 0 {
		t.Errorf("expected 0 unread entries, got %d", len(unread))
	}

	if err := s.MarkEntryUnread(ctx, entryID); err != nil {
		t.Fatalf("MarkEntryUnread failed: %v", err)
	}

	unread, _ = s.ListEntries(ctx, model.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 1 {
		t.Errorf("expected 1 unread entry, got %d", len(unread))
	}
}

func TestMarkEntriesReadUnread(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*model.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := s.MarkEntriesRead(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesRead failed: %v", err)
	}

	unread, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 0 {
		t.Errorf("expected 0 unread entries, got %d", len(unread))
	}

	if err := s.MarkEntriesUnread(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesUnread failed: %v", err)
	}

	unread, _ = s.ListEntries(ctx, model.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != 2 {
		t.Errorf("expected 2 unread entries, got %d", len(unread))
	}
}

func TestEntryStateMutationsEmptyIDs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.MarkEntriesRead(ctx, nil); err != nil {
		t.Fatalf("MarkEntriesRead(nil) failed: %v", err)
	}
	if err := s.MarkEntriesUnread(ctx, []int64{}); err != nil {
		t.Fatalf("MarkEntriesUnread(empty) failed: %v", err)
	}
	if err := s.StarEntries(ctx, nil); err != nil {
		t.Fatalf("StarEntries(nil) failed: %v", err)
	}
	if err := s.UnstarEntries(ctx, []int64{}); err != nil {
		t.Fatalf("UnstarEntries(empty) failed: %v", err)
	}
}

func TestStarUnstarEntry(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*model.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 1})
	id := entries[0].ID

	if err := s.StarEntry(ctx, id); err != nil {
		t.Fatalf("StarEntry failed: %v", err)
	}

	entries, _ = s.ListEntries(ctx, model.EntryFilter{Limit: 1})
	if entries[0].StarredAt == nil {
		t.Error("expected StarredAt to be set")
	}

	starred, _ := s.ListEntries(ctx, model.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 1 {
		t.Errorf("got %d starred, want 1", len(starred))
	}

	if err := s.UnstarEntry(ctx, id); err != nil {
		t.Fatalf("UnstarEntry failed: %v", err)
	}

	starred, _ = s.ListEntries(ctx, model.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 0 {
		t.Errorf("got %d starred after unstar, want 0", len(starred))
	}
}

func TestStarUnstarEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &model.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	_, _ = s.AddEntries(ctx, []*model.Entry{
		{FeedID: feed.ID, GUID: "1", Title: "Entry 1", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
		{FeedID: feed.ID, GUID: "2", Title: "Entry 2", Categories: json.RawMessage("[]"), Enclosures: json.RawMessage("[]"), Authors: json.RawMessage("[]"), Links: json.RawMessage("[]"), Contributors: json.RawMessage("[]")},
	})

	entries, _ := s.ListEntries(ctx, model.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := s.StarEntries(ctx, ids); err != nil {
		t.Fatalf("StarEntries failed: %v", err)
	}

	starred, _ := s.ListEntries(ctx, model.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 2 {
		t.Errorf("got %d starred, want 2", len(starred))
	}

	if err := s.UnstarEntries(ctx, ids); err != nil {
		t.Fatalf("UnstarEntries failed: %v", err)
	}

	starred, _ = s.ListEntries(ctx, model.EntryFilter{StarredOnly: true, Limit: 10})
	if len(starred) != 0 {
		t.Errorf("got %d starred after unstar, want 0", len(starred))
	}
}
