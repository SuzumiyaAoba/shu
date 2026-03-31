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

func newTestService(t *testing.T, handler http.Handler) *core.Service {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := core.New(s, logger)

	if handler != nil {
		ts := httptest.NewServer(handler)
		t.Cleanup(ts.Close)
		svc.SetHTTPClient(ts.Client())
	}

	return svc
}

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test Blog</title>
    <link>https://example.com</link>
    <description>A blog about testing</description>
    <language>en</language>
    <image>
      <url>https://example.com/logo.png</url>
      <title>Test Blog</title>
    </image>
    <item>
      <title>Post 1</title>
      <link>https://example.com/post-1</link>
      <guid>post-1</guid>
      <description>First post</description>
      <content:encoded><![CDATA[<p>Full content of post 1</p>]]></content:encoded>
      <author>alice@example.com (Alice)</author>
      <category>Go</category>
      <category>Testing</category>
      <enclosure url="https://example.com/ep1.mp3" length="12345" type="audio/mpeg"/>
    </item>
    <item>
      <title>Post 2</title>
      <link>https://example.com/post-2</link>
      <guid>post-2</guid>
      <description>Second post</description>
    </item>
  </channel>
</rss>`

func TestUserAgentHeader(t *testing.T) {
	var gotUA string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	s, _ := store.NewSQLiteStore(":memory:")
	defer s.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := core.New(s, logger)
	// Don't call SetHTTPClient - use the default client with User-Agent transport

	// The default client has User-Agent transport but uses http.DefaultTransport,
	// which won't route to our test server. We need to wrap the test server's transport.
	testClient := ts.Client()
	svc.SetHTTPClientWithUserAgent(testClient)

	_, _ = svc.AddFeed(context.Background(), ts.URL+"/feed.xml", "")
	if gotUA != "shu/0.1" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "shu/0.1")
	}
}

func TestAddFeed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	})
	svc := newTestService(t, handler)
	ctx := context.Background()

	// AddFeed needs a URL - in tests we use the test server URL
	// but AddFeed takes a URL that it fetches, so we need the test server
	ts := httptest.NewServer(handler)
	defer ts.Close()
	svc.SetHTTPClient(ts.Client())

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
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
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

func TestListFeeds(t *testing.T) {
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
		io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
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
