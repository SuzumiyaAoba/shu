package core_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestMarkReadUnread(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	_, _ = svc.FetchFeed(ctx, feed.ID)

	entries, _ := svc.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}

	id := entries[0].ID

	// Mark as read.
	if err := svc.MarkEntryRead(ctx, id); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	// Unread filter should not include it.
	unread, _ := svc.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != len(entries)-1 {
		t.Errorf("got %d unread, want %d", len(unread), len(entries)-1)
	}

	// Mark as unread.
	if err := svc.MarkEntryUnread(ctx, id); err != nil {
		t.Fatalf("MarkEntryUnread failed: %v", err)
	}

	unread, _ = svc.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != len(entries) {
		t.Errorf("got %d unread, want %d", len(unread), len(entries))
	}
}

func TestMarkEntriesReadUnread(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	_, _ = svc.FetchFeed(ctx, feed.ID)

	entries, _ := svc.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}

	if err := svc.MarkEntriesRead(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesRead failed: %v", err)
	}

	unread, _ := svc.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != len(entries)-2 {
		t.Errorf("got %d unread, want %d", len(unread), len(entries)-2)
	}

	if err := svc.MarkEntriesUnread(ctx, ids); err != nil {
		t.Fatalf("MarkEntriesUnread failed: %v", err)
	}

	unread, _ = svc.ListEntries(ctx, core.EntryFilter{Limit: 10, UnreadOnly: true})
	if len(unread) != len(entries) {
		t.Errorf("got %d unread, want %d", len(unread), len(entries))
	}
}

func TestMarkEntriesReadUnreadEmptyIDs(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	if err := svc.MarkEntriesRead(ctx, nil); err != nil {
		t.Fatalf("MarkEntriesRead(nil) failed: %v", err)
	}
	if err := svc.MarkEntriesRead(ctx, []int64{}); err != nil {
		t.Fatalf("MarkEntriesRead(empty) failed: %v", err)
	}
	if err := svc.MarkEntriesUnread(ctx, nil); err != nil {
		t.Fatalf("MarkEntriesUnread(nil) failed: %v", err)
	}
	if err := svc.MarkEntriesUnread(ctx, []int64{}); err != nil {
		t.Fatalf("MarkEntriesUnread(empty) failed: %v", err)
	}
}

func TestMarkEntryReadWrapsStoreError(t *testing.T) {
	svc := core.New(newEntryStateErrorStore(errors.New("boom")), slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := svc.MarkEntryRead(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mark read 42: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkEntriesUnreadWrapsStoreError(t *testing.T) {
	svc := core.New(newEntryStateErrorStore(errors.New("boom")), slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := svc.MarkEntriesUnread(context.Background(), []int64{1, 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mark unread entries: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type entryStateErrorStore struct {
	err error
}

func newEntryStateErrorStore(err error) *entryStateErrorStore {
	return &entryStateErrorStore{err: err}
}

func (s *entryStateErrorStore) AddFeed(context.Context, *core.Feed) error { return nil }

func (s *entryStateErrorStore) GetFeed(context.Context, int64) (*core.Feed, error) {
	return nil, nil
}

func (s *entryStateErrorStore) GetFeedByURL(context.Context, string) (*core.Feed, error) {
	return nil, nil
}

func (s *entryStateErrorStore) ListFeeds(context.Context) ([]*core.Feed, error) { return nil, nil }

func (s *entryStateErrorStore) RemoveFeed(context.Context, int64) error { return nil }

func (s *entryStateErrorStore) UpdateFeed(context.Context, int64, core.FeedUpdate) error { return nil }

func (s *entryStateErrorStore) UpdateFeedFetchedAt(context.Context, int64) error { return nil }

func (s *entryStateErrorStore) UpdateFeedCacheHeaders(context.Context, int64, string, string) error {
	return nil
}

func (s *entryStateErrorStore) RecordFeedError(context.Context, int64, string) error { return nil }

func (s *entryStateErrorStore) ResetFeedError(context.Context, int64) error { return nil }

func (s *entryStateErrorStore) SetFeedDisabled(context.Context, int64, bool) error { return nil }

func (s *entryStateErrorStore) AddEntries(context.Context, []*core.Entry) (int, error) { return 0, nil }

func (s *entryStateErrorStore) GetEntry(context.Context, int64) (*core.Entry, error) { return nil, nil }

func (s *entryStateErrorStore) ListEntries(context.Context, core.EntryFilter) ([]*core.Entry, error) {
	return nil, nil
}

func (s *entryStateErrorStore) CountEntries(context.Context, core.EntryFilter) (int, error) {
	return 0, nil
}

func (s *entryStateErrorStore) SearchEntries(context.Context, string, int) ([]*core.Entry, error) {
	return nil, nil
}

func (s *entryStateErrorStore) SearchEntriesPage(context.Context, string, int, int) ([]*core.Entry, error) {
	return nil, nil
}

func (s *entryStateErrorStore) CountSearchEntries(context.Context, string) (int, error) {
	return 0, nil
}

func (s *entryStateErrorStore) FindDuplicateEntries(context.Context, int64) ([]*core.Entry, error) {
	return nil, nil
}

func (s *entryStateErrorStore) MarkEntryRead(context.Context, int64) error { return s.err }

func (s *entryStateErrorStore) MarkEntriesRead(context.Context, []int64) error { return s.err }

func (s *entryStateErrorStore) MarkEntryUnread(context.Context, int64) error { return s.err }

func (s *entryStateErrorStore) MarkEntriesUnread(context.Context, []int64) error { return s.err }

func (s *entryStateErrorStore) StarEntry(context.Context, int64) error { return s.err }

func (s *entryStateErrorStore) StarEntries(context.Context, []int64) error { return s.err }

func (s *entryStateErrorStore) UnstarEntry(context.Context, int64) error { return s.err }

func (s *entryStateErrorStore) UnstarEntries(context.Context, []int64) error { return s.err }

func (s *entryStateErrorStore) AddTag(context.Context, int64, string) error { return nil }

func (s *entryStateErrorStore) RemoveTag(context.Context, int64, string) error { return nil }

func (s *entryStateErrorStore) ListTags(context.Context, int64) ([]core.Tag, error) { return nil, nil }

func (s *entryStateErrorStore) ListFeedTags(context.Context) (map[int64][]core.Tag, error) {
	return nil, nil
}

func (s *entryStateErrorStore) ListAllTags(context.Context) ([]core.Tag, error) { return nil, nil }

func (s *entryStateErrorStore) ListFeedsByTag(context.Context, string) ([]*core.Feed, error) {
	return nil, nil
}

func (s *entryStateErrorStore) FeedStats(context.Context) ([]core.FeedStats, error) { return nil, nil }

func (s *entryStateErrorStore) CleanupEntries(context.Context, time.Time) (int, error) { return 0, nil }

func (s *entryStateErrorStore) Close() error { return nil }
