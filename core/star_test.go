package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestStarEntries(t *testing.T) {
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
	if err := svc.StarEntries(ctx, ids); err != nil {
		t.Fatalf("StarEntries failed: %v", err)
	}

	starred, _ := svc.ListEntries(ctx, core.EntryFilter{Limit: 10, StarredOnly: true})
	if len(starred) != 2 {
		t.Errorf("got %d starred, want 2", len(starred))
	}

	if err := svc.UnstarEntries(ctx, ids); err != nil {
		t.Fatalf("UnstarEntries failed: %v", err)
	}

	starred, _ = svc.ListEntries(ctx, core.EntryFilter{Limit: 10, StarredOnly: true})
	if len(starred) != 0 {
		t.Errorf("got %d starred, want 0", len(starred))
	}
}
