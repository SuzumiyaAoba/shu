package core_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

func TestUserAgentHeader(t *testing.T) {
	var gotUA string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	s, _ := store.NewSQLiteStore(":memory:")
	defer func() { _ = s.Close() }()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testClient := ts.Client()
	svc := core.New(s, logger, core.WithAllowPrivateAddresses(true), core.WithHTTPClientWithUserAgent(testClient))

	_, _ = svc.AddFeed(context.Background(), ts.URL+"/feed.xml", "")
	if gotUA != "shu/0.1" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "shu/0.1")
	}
}

func TestAddFeed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ctx := context.Background()

	ts := httptest.NewServer(handler)
	defer ts.Close()
	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))

	feed, err := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}
	if feed.Title != "Test Blog" {
		t.Errorf("Title = %q, want %q", feed.Title, "Test Blog")
	}
	if feed.URL != ts.URL+"/feed.xml" {
		t.Errorf("URL = %q, want %q", feed.URL, ts.URL+"/feed.xml")
	}
	if feed.SiteURL != "https://example.com" {
		t.Errorf("SiteURL = %q, want %q", feed.SiteURL, "https://example.com")
	}
	if feed.Description != "A blog about testing" {
		t.Errorf("Description = %q, want %q", feed.Description, "A blog about testing")
	}
	if feed.Language != "en" {
		t.Errorf("Language = %q, want %q", feed.Language, "en")
	}
	if feed.ImageURL != "https://example.com/logo.png" {
		t.Errorf("ImageURL = %q, want %q", feed.ImageURL, "https://example.com/logo.png")
	}
	if feed.FeedType != "rss" {
		t.Errorf("FeedType = %q, want %q", feed.FeedType, "rss")
	}
}

func TestAddFeedWithTitleOverride(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, err := svc.AddFeed(ctx, ts.URL+"/feed.xml", "My Custom Title")
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}
	if feed.Title != "My Custom Title" {
		t.Errorf("Title = %q, want %q", feed.Title, "My Custom Title")
	}
}

func TestAddFeedInvalidURL(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	_, err := svc.AddFeed(ctx, "not-a-url", "")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestAddFeedInvalidDocument(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>not a feed</html>")
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	_, err := svc.AddFeed(ctx, ts.URL+"/index.html", "")
	if err == nil {
		t.Fatal("expected invalid feed error")
	}
	if !errors.Is(err, core.ErrInvalidFeed) {
		t.Fatalf("expected ErrInvalidFeed, got %v", err)
	}
}

func TestListFeeds(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	_, _ = svc.AddFeed(ctx, ts.URL+"/feed1.xml", "")
	_, _ = svc.AddFeed(ctx, ts.URL+"/feed2.xml", "")

	feeds, err := svc.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("got %d feeds, want 2", len(feeds))
	}
}

func TestRemoveFeed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, _ := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	err := svc.RemoveFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("RemoveFeed failed: %v", err)
	}

	feeds, _ := svc.ListFeeds(ctx)
	if len(feeds) != 0 {
		t.Errorf("got %d feeds, want 0", len(feeds))
	}
}

func TestNewWithNilLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = s.Close() }()

	svc := core.New(s, nil, core.WithAllowPrivateAddresses(true), core.WithHTTPClient(ts.Client()))

	ctx := context.Background()
	if _, err := svc.AddFeed(ctx, ts.URL+"/feed.xml", ""); err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}
}
