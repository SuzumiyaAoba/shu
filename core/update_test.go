package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/model"
)

func TestUpdateFeed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")

	newTitle := "Updated Title"
	if err := svc.UpdateFeed(ctx, feed.ID, model.FeedUpdate{Title: &newTitle}); err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	feeds, _ := svc.ListFeeds(ctx)
	if len(feeds) != 1 || feeds[0].Title != "Updated Title" {
		t.Errorf("title = %q, want %q", feeds[0].Title, "Updated Title")
	}
}
