package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

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

func TestFeedStatsNoEntries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Example"}
	_ = s.AddFeed(ctx, feed)

	stats, err := s.FeedStats(ctx)
	if err != nil {
		t.Fatalf("FeedStats failed: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("got %d stats, want 1", len(stats))
	}
	if stats[0].TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", stats[0].TotalCount)
	}
	if stats[0].UnreadCount != 0 {
		t.Errorf("UnreadCount = %d, want 0", stats[0].UnreadCount)
	}
	if stats[0].StarredCount != 0 {
		t.Errorf("StarredCount = %d, want 0", stats[0].StarredCount)
	}
}
