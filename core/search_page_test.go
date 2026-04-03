package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestSearchEntriesPage(t *testing.T) {
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

	page, err := svc.SearchEntriesPage(ctx, "Post", 1, 0)
	if err != nil {
		t.Fatalf("SearchEntriesPage failed: %v", err)
	}
	if page.TotalCount != 2 {
		t.Fatalf("TotalCount = %d, want 2", page.TotalCount)
	}
	if len(page.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(page.Entries))
	}
	if !page.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if page.Limit != 1 || page.Offset != 0 {
		t.Fatalf("unexpected page metadata: %+v", page)
	}
}

func TestSearchEntriesPageDefaultLimit(t *testing.T) {
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

	page, err := svc.SearchEntriesPage(ctx, "Post", 0, 0)
	if err != nil {
		t.Fatalf("SearchEntriesPage failed: %v", err)
	}
	if page.Limit != len(page.Entries) {
		t.Fatalf("Limit = %d, want %d", page.Limit, len(page.Entries))
	}
	if page.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
}
