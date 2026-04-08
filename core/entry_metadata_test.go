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

	categories, err := model.ParseCategories(entry)
	if err != nil {
		t.Fatalf("ParseCategories failed: %v", err)
	}
	if len(categories) != 2 || categories[0].Term != "golang" {
		t.Fatalf("unexpected categories: %+v", categories)
	}

	authors, err := model.ParseAuthors(entry)
	if err != nil {
		t.Fatalf("ParseAuthors failed: %v", err)
	}
	if len(authors) != 2 || authors[0].URI != "https://alice.example.com" {
		t.Fatalf("unexpected authors: %+v", authors)
	}

	links, err := model.ParseLinks(entry)
	if err != nil {
		t.Fatalf("ParseLinks failed: %v", err)
	}
	if len(links) != 2 || links[0].Href == "" {
		t.Fatalf("unexpected links: %+v", links)
	}

	contributors, err := model.ParseContributors(entry)
	if err != nil {
		t.Fatalf("ParseContributors failed: %v", err)
	}
	if len(contributors) != 1 || contributors[0].Name != "Charlie" {
		t.Fatalf("unexpected contributors: %+v", contributors)
	}

	source, err := model.ParseSource(entry)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if source == nil || source.Title != "Original Source" {
		t.Fatalf("unexpected source: %+v", source)
	}
}

func TestEntryMetadataHelpersEmptyValues(t *testing.T) {
	entry := &model.Entry{}

	categories, err := model.ParseCategories(entry)
	if err != nil {
		t.Fatalf("ParseCategories failed: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("expected empty categories, got %+v", categories)
	}

	source, err := model.ParseSource(entry)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil source, got %+v", source)
	}
}
