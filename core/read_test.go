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
