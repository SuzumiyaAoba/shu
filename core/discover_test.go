package core_test

import (
	"context"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestDiscoverFeeds(t *testing.T) {
	htmlPage := `<!DOCTYPE html>
<html>
<head>
  <title>My Blog</title>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml" title="RSS">
  <link rel="alternate" type="application/atom+xml" href="https://example.com/atom.xml" title="Atom">
  <link rel="stylesheet" href="/style.css">
</head>
<body>Hello</body>
</html>`

	ts := newHTMLTestServer(t, htmlPage)

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feeds, err := svc.DiscoverFeeds(ctx, ts.URL+"/")
	if err != nil {
		t.Fatalf("DiscoverFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Fatalf("got %d feeds, want 2", len(feeds))
	}

	// First feed should be relative, resolved to test server URL.
	if feeds[0] != ts.URL+"/feed.xml" {
		t.Errorf("feeds[0] = %q, expected relative URL resolved", feeds[0])
	}
	// Second feed should be absolute.
	if feeds[1] != "https://example.com/atom.xml" {
		t.Errorf("feeds[1] = %q, want %q", feeds[1], "https://example.com/atom.xml")
	}
}

func TestDiscoverFeedsNone(t *testing.T) {
	htmlPage := `<!DOCTYPE html><html><head><title>No feeds</title></head><body>Hello</body></html>`

	ts := newHTMLTestServer(t, htmlPage)

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))

	feeds, err := svc.DiscoverFeeds(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("DiscoverFeeds failed: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("got %d feeds, want 0", len(feeds))
	}
}
