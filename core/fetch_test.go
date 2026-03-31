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

const testAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test Blog</title>
  <link href="https://example.com" rel="alternate"/>
  <link href="https://example.com/atom.xml" rel="self"/>
  <id>urn:uuid:feed-1</id>
  <updated>2026-03-01T12:00:00Z</updated>
  <entry>
    <title>Atom Post 1</title>
    <id>urn:uuid:entry-1</id>
    <link href="https://example.com/atom-post-1" rel="alternate" type="text/html" hreflang="en"/>
    <link href="https://example.com/atom-post-1.pdf" rel="enclosure" type="application/pdf" length="9999"/>
    <updated>2026-03-01T12:00:00Z</updated>
    <published>2026-02-28T10:00:00Z</published>
    <summary>Atom summary</summary>
    <content type="html">&lt;p&gt;Full Atom content&lt;/p&gt;</content>
    <author>
      <name>Alice</name>
      <email>alice@example.com</email>
      <uri>https://alice.example.com</uri>
    </author>
    <author>
      <name>Bob</name>
      <email>bob@example.com</email>
    </author>
    <contributor>
      <name>Charlie</name>
      <email>charlie@example.com</email>
      <uri>https://charlie.example.com</uri>
    </contributor>
    <category term="golang" scheme="https://example.com/tags" label="Go Language"/>
    <category term="atom"/>
    <rights>CC BY 4.0</rights>
    <source>
      <title>Original Source</title>
      <id>urn:uuid:source-1</id>
      <updated>2026-01-01T00:00:00Z</updated>
    </source>
  </entry>
  <entry>
    <title>Atom Post 2</title>
    <id>urn:uuid:entry-2</id>
    <link href="https://example.com/atom-post-2" rel="alternate"/>
    <updated>2026-03-01T11:00:00Z</updated>
    <summary>Simple atom entry</summary>
  </entry>
</feed>`

func TestFetchFeedAtomFields(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		io.WriteString(w, testAtomFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestService(t, nil)
	svc.SetHTTPClient(ts.Client())
	ctx := context.Background()

	feed, err := svc.AddFeed(ctx, ts.URL+"/atom.xml", "")
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}
	if feed.FeedType != "atom" {
		t.Errorf("FeedType = %q, want %q", feed.FeedType, "atom")
	}

	entries, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	e := entries[0]

	// Content
	if e.Content != "<p>Full Atom content</p>" {
		t.Errorf("Content = %q", e.Content)
	}

	// Author (convenience field = first author name)
	if e.Author != "Alice" {
		t.Errorf("Author = %q, want %q", e.Author, "Alice")
	}

	// Authors (full structured JSON with URI)
	if e.Authors == "[]" {
		t.Error("expected Authors to be populated")
	}
	// Should contain Alice's URI from Atom-specific parse
	if !contains(e.Authors, "alice@example.com") || !contains(e.Authors, "https://alice.example.com") {
		t.Errorf("Authors = %s, expected Alice with email and URI", e.Authors)
	}
	if !contains(e.Authors, "Bob") {
		t.Errorf("Authors = %s, expected Bob", e.Authors)
	}

	// Links (full structured with rel, type, hreflang)
	if e.Links == "[]" {
		t.Error("expected Links to be populated")
	}
	if !contains(e.Links, "alternate") || !contains(e.Links, "text/html") {
		t.Errorf("Links = %s, expected alternate link with type", e.Links)
	}

	// Categories (structured with term/scheme/label)
	if e.Categories == "[]" {
		t.Error("expected Categories to be populated")
	}
	if !contains(e.Categories, "golang") || !contains(e.Categories, "https://example.com/tags") {
		t.Errorf("Categories = %s, expected structured category with scheme", e.Categories)
	}

	// Contributors
	if e.Contributors == "[]" {
		t.Error("expected Contributors to be populated")
	}
	if !contains(e.Contributors, "Charlie") || !contains(e.Contributors, "https://charlie.example.com") {
		t.Errorf("Contributors = %s, expected Charlie with URI", e.Contributors)
	}

	// Rights
	if e.Rights != "CC BY 4.0" {
		t.Errorf("Rights = %q, want %q", e.Rights, "CC BY 4.0")
	}

	// Source
	if e.Source == "" {
		t.Error("expected Source to be populated")
	}
	if !contains(e.Source, "Original Source") || !contains(e.Source, "urn:uuid:source-1") {
		t.Errorf("Source = %s, expected source with title and id", e.Source)
	}

	// Entry 2: minimal Atom entry, should have defaults.
	e2 := entries[1]
	if e2.Contributors != "[]" {
		t.Errorf("e2.Contributors = %q, want %q", e2.Contributors, "[]")
	}
	if e2.Rights != "" {
		t.Errorf("e2.Rights = %q, want empty", e2.Rights)
	}
	if e2.Source != "" {
		t.Errorf("e2.Source = %q, want empty", e2.Source)
	}
}

// contains checks if s contains the substring sub.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFetchFeedNotFound(t *testing.T) {
	svc := newTestService(t, nil)
	ctx := context.Background()

	_, err := svc.FetchFeed(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}
}
