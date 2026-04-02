package core_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestListEntriesPage(t *testing.T) {
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

	page, err := svc.ListEntriesPage(ctx, core.EntryFilter{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntriesPage failed: %v", err)
	}
	if page.TotalCount != 2 {
		t.Fatalf("TotalCount = %d, want 2", page.TotalCount)
	}
	if len(page.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(page.Entries))
	}
	if !page.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if page.Limit != 1 || page.Offset != 0 {
		t.Fatalf("unexpected page metadata: %+v", page)
	}
}
