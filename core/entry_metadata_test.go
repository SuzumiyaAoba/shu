package core_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestEntryMetadataHelpers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = io.WriteString(w, testAtomFeed)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	svc := newTestServiceWithOptions(t, nil, core.WithHTTPClient(ts.Client()))
	ctx := context.Background()

	feed, err := svc.AddFeed(ctx, ts.URL+"/atom.xml", "")
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	entries, err := svc.FetchFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}

	entry := entries[0]

	categories, err := entry.ParseCategories()
	if err != nil {
		t.Fatalf("ParseCategories failed: %v", err)
	}
	if len(categories) != 2 || categories[0].Term != "golang" {
		t.Fatalf("unexpected categories: %+v", categories)
	}

	authors, err := entry.ParseAuthors()
	if err != nil {
		t.Fatalf("ParseAuthors failed: %v", err)
	}
	if len(authors) != 2 || authors[0].URI != "https://alice.example.com" {
		t.Fatalf("unexpected authors: %+v", authors)
	}

	links, err := entry.ParseLinks()
	if err != nil {
		t.Fatalf("ParseLinks failed: %v", err)
	}
	if len(links) != 2 || links[0].Href == "" {
		t.Fatalf("unexpected links: %+v", links)
	}

	contributors, err := entry.ParseContributors()
	if err != nil {
		t.Fatalf("ParseContributors failed: %v", err)
	}
	if len(contributors) != 1 || contributors[0].Name != "Charlie" {
		t.Fatalf("unexpected contributors: %+v", contributors)
	}

	source, err := entry.ParseSource()
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if source == nil || source.Title != "Original Source" {
		t.Fatalf("unexpected source: %+v", source)
	}
}

func TestEntryMetadataHelpersEmptyValues(t *testing.T) {
	entry := &core.Entry{}

	categories, err := entry.ParseCategories()
	if err != nil {
		t.Fatalf("ParseCategories failed: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("expected empty categories, got %+v", categories)
	}

	source, err := entry.ParseSource()
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil source, got %+v", source)
	}
}

func TestEntryMetadataHelpersUseCache(t *testing.T) {
	entry := &core.Entry{
		Categories: json.RawMessage(`[{"term":"go"}]`),
		Source:     json.RawMessage(`{"title":"cached","id":"1","updated":"2026-01-01T00:00:00Z"}`),
	}

	categories, err := entry.ParseCategories()
	if err != nil {
		t.Fatalf("ParseCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("unexpected categories: %+v", categories)
	}

	entry.Categories = json.RawMessage(`invalid json`)
	categories, err = entry.ParseCategories()
	if err != nil {
		t.Fatalf("expected cached ParseCategories result, got error: %v", err)
	}
	if len(categories) != 1 || categories[0].Term != "go" {
		t.Fatalf("unexpected cached categories: %+v", categories)
	}

	source, err := entry.ParseSource()
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if source == nil || source.Title != "cached" {
		t.Fatalf("unexpected source: %+v", source)
	}

	entry.Source = json.RawMessage(`invalid json`)
	source, err = entry.ParseSource()
	if err != nil {
		t.Fatalf("expected cached ParseSource result, got error: %v", err)
	}
	if source == nil || source.Title != "cached" {
		t.Fatalf("unexpected cached source: %+v", source)
	}
}
