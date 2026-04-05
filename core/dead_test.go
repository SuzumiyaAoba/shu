package core_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

func TestListDeadFeeds(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	st, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := core.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)), core.WithAllowPrivateAddresses(true), core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	deadFeed, _ := svc.AddFeed(ctx, ts.URL+"/dead.xml", "Dead Feed")
	errFeed, _ := svc.AddFeed(ctx, ts.URL+"/err.xml", "Err Feed")
	manualFeed, _ := svc.AddFeed(ctx, ts.URL+"/manual.xml", "Manual Feed")
	for i := 0; i < 5; i++ {
		_ = st.RecordFeedError(ctx, deadFeed.ID, "timeout")
	}
	_ = st.RecordFeedError(ctx, errFeed.ID, "timeout")
	_ = svc.DisableFeed(ctx, manualFeed.ID)

	dead, err := svc.ListDeadFeeds(ctx)
	if err != nil {
		t.Fatalf("ListDeadFeeds failed: %v", err)
	}
	if len(dead) != 2 {
		t.Fatalf("dead feeds = %+v, want 2 feeds", dead)
	}
	if dead[0].ID != deadFeed.ID || dead[1].ID != errFeed.ID {
		t.Fatalf("dead feeds = %+v, want [%d %d]", dead, deadFeed.ID, errFeed.ID)
	}
}

func TestRemoveDeadFeeds(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	st, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := core.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)), core.WithAllowPrivateAddresses(true), core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	deadFeed, _ := svc.AddFeed(ctx, ts.URL+"/dead.xml", "Dead Feed")
	errFeed, _ := svc.AddFeed(ctx, ts.URL+"/err.xml", "Err Feed")
	aliveFeed, _ := svc.AddFeed(ctx, ts.URL+"/alive.xml", "Alive Feed")

	for i := 0; i < 5; i++ {
		_ = st.RecordFeedError(ctx, deadFeed.ID, "timeout")
	}
	_ = st.RecordFeedError(ctx, errFeed.ID, "timeout")

	removed, err := svc.RemoveDeadFeeds(ctx)
	if err != nil {
		t.Fatalf("RemoveDeadFeeds failed: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %+v, want 2 feeds", removed)
	}

	feeds, err := svc.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 1 || feeds[0].ID != aliveFeed.ID {
		t.Fatalf("feeds = %+v, want alive feed only", feeds)
	}
}
