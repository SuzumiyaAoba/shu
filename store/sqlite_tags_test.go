package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestTags(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feed1 := &core.Feed{URL: "https://a.com/feed", Title: "A"}
	feed2 := &core.Feed{URL: "https://b.com/feed", Title: "B"}
	_ = s.AddFeed(ctx, feed1)
	_ = s.AddFeed(ctx, feed2)

	if err := s.AddTag(ctx, feed1.ID, "tech"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if err := s.AddTag(ctx, feed1.ID, "go"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if err := s.AddTag(ctx, feed2.ID, "tech"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	tags, err := s.ListTags(ctx, feed1.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("got %d tags, want 2", len(tags))
	}

	allTags, err := s.ListAllTags(ctx)
	if err != nil {
		t.Fatalf("ListAllTags failed: %v", err)
	}
	if len(allTags) != 2 {
		t.Errorf("got %d tags, want 2", len(allTags))
	}

	techFeeds, err := s.ListFeedsByTag(ctx, "tech")
	if err != nil {
		t.Fatalf("ListFeedsByTag failed: %v", err)
	}
	if len(techFeeds) != 2 {
		t.Errorf("got %d feeds with 'tech' tag, want 2", len(techFeeds))
	}

	if err := s.RemoveTag(ctx, feed1.ID, "go"); err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}
	tags, _ = s.ListTags(ctx, feed1.ID)
	if len(tags) != 1 {
		t.Errorf("got %d tags after remove, want 1", len(tags))
	}

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
