package core_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

// newTestServiceNoHTTP creates a service that is not expected to make any HTTP
// requests during OPML import tests.
func newTestServiceNoHTTP(t *testing.T) *core.Service {
	t.Helper()
	return newTestService(t, nil)
}

func TestExportOPML(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	_, _ = svc.AddFeed(ctx, ts.URL+"/feed1.xml", "Feed One")
	_, _ = svc.AddFeed(ctx, ts.URL+"/feed2.xml", "Feed Two")

	opml, err := svc.ExportOPML(ctx)
	if err != nil {
		t.Fatalf("ExportOPML failed: %v", err)
	}

	if opml.Version != "2.0" {
		t.Errorf("Version = %q, want %q", opml.Version, "2.0")
	}
	if len(opml.Body.Outlines) != 2 {
		t.Errorf("got %d outlines, want 2", len(opml.Body.Outlines))
	}
	if opml.Body.Outlines[0].Text != "Feed One" {
		t.Errorf("first outline text = %q", opml.Body.Outlines[0].Text)
	}
}

func TestExportOPMLWithTags(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	f1, _ := svc.AddFeed(ctx, ts.URL+"/feed1.xml", "Tagged Feed")
	_, _ = svc.AddFeed(ctx, ts.URL+"/feed2.xml", "Untagged Feed")
	_ = svc.AddTag(ctx, f1.ID, "tech")

	opml, err := svc.ExportOPML(ctx)
	if err != nil {
		t.Fatalf("ExportOPML failed: %v", err)
	}

	// Should have 1 untagged feed at top level + 1 "tech" category group.
	if len(opml.Body.Outlines) != 2 {
		t.Fatalf("got %d top-level outlines, want 2", len(opml.Body.Outlines))
	}

	// First should be the untagged feed.
	if opml.Body.Outlines[0].XMLURL == "" {
		t.Error("expected first outline to be a feed (untagged)")
	}

	// Second should be the "tech" group with 1 child.
	group := opml.Body.Outlines[1]
	if group.Text != "tech" {
		t.Errorf("group text = %q, want %q", group.Text, "tech")
	}
	if len(group.Outlines) != 1 {
		t.Errorf("got %d feeds in tech group, want 1", len(group.Outlines))
	}
}

func TestImportOPML(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Feed 1" type="rss" xmlUrl="https://example.com/feed1.xml"/>
    <outline text="Tech" title="Tech">
      <outline text="Feed 2" type="rss" xmlUrl="https://example.com/feed2.xml"/>
    </outline>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}

	// Feed 2 should have "Tech" tag.
	feeds, _ := svc.ListFeedsByTag(ctx, "Tech")
	if len(feeds) != 1 {
		t.Errorf("got %d feeds with Tech tag, want 1", len(feeds))
	}
}

func TestImportOPMLNestedCategoriesAddAllTags(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Go">
        <outline text="Feed 1" type="rss" xmlUrl="https://example.com/feed1.xml"/>
      </outline>
    </outline>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}

	feed, err := svc.GetFeed(ctx, 1)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	tags, err := svc.ListTags(ctx, feed.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	if tags[0].Name != "Go" || tags[1].Name != "Tech" {
		t.Fatalf("tags = %+v, want [Go Tech]", tags)
	}
}

func TestImportOPMLDuplicateSkip(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	// Add the feed first via AddFeedDirect.
	_ = svc.AddFeedDirect(ctx, &core.Feed{URL: "https://example.com/feed.xml", Title: "Existing"})

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Same Feed" type="rss" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if added != 0 {
		t.Errorf("added = %d, want 0 (duplicate should be skipped)", added)
	}
}

func TestImportOPMLDuplicateAddsTag(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	feed := &core.Feed{URL: "https://example.com/feed.xml", Title: "Existing"}
	if err := svc.AddFeedDirect(ctx, feed); err != nil {
		t.Fatalf("AddFeedDirect failed: %v", err)
	}

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Same Feed" type="rss" xmlUrl="https://example.com/feed.xml"/>
    </outline>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if added != 0 {
		t.Fatalf("added = %d, want 0", added)
	}

	tags, err := svc.ListTags(ctx, feed.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "Tech" {
		t.Fatalf("tags = %+v, want [Tech]", tags)
	}
}

func TestImportOPMLEmptyURLSkipped(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="No URL outline"/>
    <outline text="Feed 1" type="rss" xmlUrl="https://example.com/feed1.xml"/>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if added != 1 {
		t.Errorf("added = %d, want 1 (no-URL outline skipped)", added)
	}
}

func TestImportOPMLDetailed(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	// Pre-register a feed so it shows as reused.
	_ = svc.AddFeedDirect(ctx, &core.Feed{URL: "https://example.com/feed1.xml", Title: "Existing"})

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Existing Feed" type="rss" xmlUrl="https://example.com/feed1.xml"/>
      <outline text="New Feed" type="rss" xmlUrl="https://example.com/feed2.xml"/>
    </outline>
  </body>
</opml>`

	result, err := svc.ImportOPMLDetailed(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPMLDetailed failed: %v", err)
	}
	if result.AddedCount != 1 {
		t.Fatalf("AddedCount = %d, want 1", result.AddedCount)
	}
	if result.ReusedCount != 1 {
		t.Fatalf("ReusedCount = %d, want 1", result.ReusedCount)
	}
	if result.TaggedCount != 2 {
		t.Fatalf("TaggedCount = %d, want 2", result.TaggedCount)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("Issues = %+v, want none", result.Issues)
	}
}

func TestImportOPMLInvalidDocument(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	_, err := svc.ImportOPML(ctx, strings.NewReader("<opml"))
	if err == nil {
		t.Fatal("expected invalid OPML error")
	}
	if !errors.Is(err, core.ErrInvalidOPML) {
		t.Fatalf("expected ErrInvalidOPML, got %v", err)
	}
}

func TestImportOPMLTitleFallback(t *testing.T) {
	svc := newTestServiceNoHTTP(t)
	ctx := context.Background()

	// title attr takes precedence over text attr; text is used as fallback.
	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="text-only" type="rss" xmlUrl="https://example.com/feed1.xml"/>
    <outline text="text-val" title="title-val" type="rss" xmlUrl="https://example.com/feed2.xml"/>
  </body>
</opml>`

	_, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}

	feeds, _ := svc.ListFeeds(ctx)
	if len(feeds) != 2 {
		t.Fatalf("got %d feeds, want 2", len(feeds))
	}
	if feeds[0].Title != "text-only" {
		t.Errorf("feed[0].Title = %q, want %q", feeds[0].Title, "text-only")
	}
	if feeds[1].Title != "title-val" {
		t.Errorf("feed[1].Title = %q, want %q", feeds[1].Title, "title-val")
	}
}
