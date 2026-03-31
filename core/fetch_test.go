package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestFetchFeed(t *testing.T) {
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

	entries, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
	if entries[0].Title != "Post 1" {
		t.Errorf("first entry title = %q, want %q", entries[0].Title, "Post 1")
	}
}

func TestFetchFeedDeduplication(t *testing.T) {
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

	// First fetch
	entries1, _ := svc.FetchFeed(ctx, feed.ID)
	// Second fetch - same items, should deduplicate
	entries2, _ := svc.FetchFeed(ctx, feed.ID)

	if len(entries1) != 2 {
		t.Errorf("first fetch: got %d entries, want 2", len(entries1))
	}
	if len(entries2) != 0 {
		t.Errorf("second fetch: got %d new entries, want 0", len(entries2))
	}
}

func TestFetchAll(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
	ctx := context.Background()

	_, _ = svc.AddFeed(ctx, ts.URL+"/feed1.xml", "")
	_, _ = svc.AddFeed(ctx, ts.URL+"/feed2.xml", "")

	count, err := svc.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll failed: %v", err)
	}
	// 2 feeds * 2 entries each = 4
	if count != 4 {
		t.Errorf("FetchAll returned %d, want 4", count)
	}
}

func TestListEntries(t *testing.T) {
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

	entries, err := svc.ListEntries(ctx, core.EntryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestFetchFeedExpandedFields(t *testing.T) {
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
	entries, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// Post 1 has content, author, categories, and enclosure.
	e := entries[0]
	if e.Content != "<p>Full content of post 1</p>" {
		t.Errorf("Content = %q, want %q", e.Content, "<p>Full content of post 1</p>")
	}
	if e.Categories == "[]" {
		t.Error("expected categories to be populated for Post 1")
	}
	if e.Enclosures == "[]" {
		t.Error("expected enclosures to be populated for Post 1")
	}

	// Post 2 has no extra fields — defaults should apply.
	e2 := entries[1]
	if e2.Content != "" {
		t.Errorf("Content = %q, want empty", e2.Content)
	}
	if e2.Categories != "[]" {
		t.Errorf("Categories = %q, want %q", e2.Categories, "[]")
	}
	if e2.Enclosures != "[]" {
		t.Errorf("Enclosures = %q, want %q", e2.Enclosures, "[]")
	}
}

func TestFetchFeedNotFound(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	_, err := svc.FetchFeed(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}
}
