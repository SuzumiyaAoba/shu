package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFetchFeedConditionalGET(t *testing.T) {
	var reqCount atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)

		// First request: return feed with ETag.
		if r.Header.Get("If-None-Match") == "" {
			w.Header().Set("ETag", `"etag-123"`)
			w.Header().Set("Content-Type", "application/rss+xml")
			io.WriteString(w, testRSSFeed)
			return
		}

		// Subsequent requests with matching ETag: 304.
		if r.Header.Get("If-None-Match") == `"etag-123"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")

	// First fetch: should get entries.
	entries, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("first FetchFeed failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("first fetch: got %d entries, want 2", len(entries))
	}

	// Second fetch: should get 304, no new entries.
	entries2, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("second FetchFeed failed: %v", err)
	}
	if len(entries2) != 0 {
		t.Errorf("second fetch: got %d entries, want 0 (304)", len(entries2))
	}
}
