package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
