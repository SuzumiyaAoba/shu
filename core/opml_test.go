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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Feed 1" type="rss" xmlUrl="` + ts.URL + `/feed1.xml"/>
    <outline text="Tech" title="Tech">
      <outline text="Feed 2" type="rss" xmlUrl="` + ts.URL + `/feed2.xml"/>
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Go">
        <outline text="Feed 1" type="rss" xmlUrl="` + ts.URL + `/feed1.xml"/>
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	// Add the feed first.
	_, _ = svc.AddFeed(ctx, ts.URL+"/feed.xml", "")

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Same Feed" type="rss" xmlUrl="` + ts.URL + `/feed.xml"/>
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, err := svc.AddFeed(ctx, ts.URL+"/feed.xml", "")
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Same Feed" type="rss" xmlUrl="` + ts.URL + `/feed.xml"/>
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

func TestImportOPMLReturnsAddFeedError(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Broken Feed" type="rss" xmlUrl="https://[invalid-url"/>
  </body>
</opml>`

	added, err := svc.ImportOPML(ctx, strings.NewReader(opmlDoc))
	if err == nil {
		t.Fatal("expected ImportOPML to return add error")
	}
	if added != 0 {
		t.Fatalf("added = %d, want 0", added)
	}
	if errors.Is(err, core.ErrFeedAlreadyExists) {
		t.Fatalf("expected non-duplicate error, got %v", err)
	}
}

func TestImportOPMLDetailed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	_, _ = svc.AddFeed(ctx, ts.URL+"/feed1.xml", "")

	opmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Existing Feed" type="rss" xmlUrl="` + ts.URL + `/feed1.xml"/>
      <outline text="New Feed" type="rss" xmlUrl="` + ts.URL + `/feed2.xml"/>
      <outline text="Broken Feed" type="rss" xmlUrl="https://[invalid-url"/>
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
	if len(result.Issues) != 1 {
		t.Fatalf("Issues = %+v, want 1 issue", result.Issues)
	}
}

func TestImportOPMLInvalidDocument(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	_, err := svc.ImportOPML(ctx, strings.NewReader("<opml"))
	if err == nil {
		t.Fatal("expected invalid OPML error")
	}
	if !errors.Is(err, core.ErrInvalidOPML) {
		t.Fatalf("expected ErrInvalidOPML, got %v", err)
	}
}
