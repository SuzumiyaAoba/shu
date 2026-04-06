package core_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/model"
)

func TestStarEntries(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	_, _ = svc.FetchFeed(ctx, feed.ID)

	entries, _ := svc.ListEntries(ctx, model.EntryFilter{Limit: 10})
	if len(entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := svc.StarEntries(ctx, ids); err != nil {
		t.Fatalf("StarEntries failed: %v", err)
	}

	starred, _ := svc.ListEntries(ctx, model.EntryFilter{Limit: 10, StarredOnly: true})
	if len(starred) != 2 {
		t.Errorf("got %d starred, want 2", len(starred))
	}

	if err := svc.UnstarEntries(ctx, ids); err != nil {
		t.Fatalf("UnstarEntries failed: %v", err)
	}

	starred, _ = svc.ListEntries(ctx, model.EntryFilter{Limit: 10, StarredOnly: true})
	if len(starred) != 0 {
		t.Errorf("got %d starred, want 0", len(starred))
	}
}

func TestStarEntriesEmptyIDs(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	if err := svc.StarEntries(ctx, nil); err != nil {
		t.Fatalf("StarEntries(nil) failed: %v", err)
	}
	if err := svc.StarEntries(ctx, []int64{}); err != nil {
		t.Fatalf("StarEntries(empty) failed: %v", err)
	}
	if err := svc.UnstarEntries(ctx, nil); err != nil {
		t.Fatalf("UnstarEntries(nil) failed: %v", err)
	}
	if err := svc.UnstarEntries(ctx, []int64{}); err != nil {
		t.Fatalf("UnstarEntries(empty) failed: %v", err)
	}
}

func TestUnstarEntriesWrapsStoreError(t *testing.T) {
	svc := core.New(newEntryStateErrorStore(io.EOF), slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := svc.UnstarEntries(context.Background(), []int64{1, 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "unstar entries: EOF" {
		t.Fatalf("unexpected error: %v", err)
	}
}
