package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestTagOperations(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")

	// Add tag.
	if err := svc.AddTag(ctx, feed.ID, "tech"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	// List tags.
	tags, err := svc.ListTags(ctx, feed.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "tech" {
		t.Errorf("tags = %v, want [tech]", tags)
	}

	// List feeds by tag.
	feeds, err := svc.ListFeedsByTag(ctx, "tech")
	if err != nil {
		t.Fatalf("ListFeedsByTag failed: %v", err)
	}
	if len(feeds) != 1 {
		t.Errorf("got %d feeds, want 1", len(feeds))
	}

	// Remove tag.
	if err := svc.RemoveTag(ctx, feed.ID, "tech"); err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}
	tags, _ = svc.ListTags(ctx, feed.ID)
	if len(tags) != 0 {
		t.Errorf("got %d tags after removal, want 0", len(tags))
	}
}
