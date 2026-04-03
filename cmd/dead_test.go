package cmd

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

func TestRemoveDeadFeedsCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	}))
	defer ts.Close()

	st, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := core.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SetHTTPClient(ts.Client())
	setTestService(t, svc)

	ctx := t.Context()
	deadFeed, _ := svc.AddFeed(ctx, ts.URL+"/dead.xml", "Dead Feed")
	errFeed, _ := svc.AddFeed(ctx, ts.URL+"/err.xml", "Err Feed")
	manualFeed, _ := svc.AddFeed(ctx, ts.URL+"/manual.xml", "Manual Feed")
	for i := 0; i < 5; i++ {
		_ = st.RecordFeedError(ctx, deadFeed.ID, "timeout")
	}
	_ = st.RecordFeedError(ctx, errFeed.ID, "timeout")
	_ = svc.DisableFeed(ctx, manualFeed.ID)

	out, err := executeCommand("remove-dead-feeds")
	if err != nil {
		t.Fatalf("remove-dead-feeds failed: %v", err)
	}
	if !strings.Contains(out, "Removed 2 dead feeds") {
		t.Fatalf("unexpected output: %s", out)
	}

	listOut, _ := executeCommand("list")
	if strings.Contains(listOut, "Dead Feed") {
		t.Fatalf("dead feed should be removed: %s", listOut)
	}
	if strings.Contains(listOut, "Err Feed") {
		t.Fatalf("err feed should be removed: %s", listOut)
	}
	if !strings.Contains(listOut, "Manual Feed") {
		t.Fatalf("manual disabled feed should remain: %s", listOut)
	}
}

func TestRemoveDeadFeedsDryRunCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	}))
	defer ts.Close()

	st, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := core.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SetHTTPClient(ts.Client())
	setTestService(t, svc)

	ctx := t.Context()
	deadFeed, _ := svc.AddFeed(ctx, ts.URL+"/dead.xml", "Dead Feed")
	errFeed, _ := svc.AddFeed(ctx, ts.URL+"/err.xml", "Err Feed")
	for i := 0; i < 5; i++ {
		_ = st.RecordFeedError(ctx, deadFeed.ID, "timeout")
	}
	_ = st.RecordFeedError(ctx, errFeed.ID, "timeout")

	out, err := executeCommand("remove-dead-feeds", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("remove-dead-feeds --dry-run --json failed: %v", err)
	}
	if !strings.Contains(out, "Dead Feed") || !strings.Contains(out, "Err Feed") {
		t.Fatalf("expected dead feed in output: %s", out)
	}

	listOut, _ := executeCommand("list")
	if !strings.Contains(listOut, "Dead Feed") {
		t.Fatalf("dead feed should remain after dry-run: %s", listOut)
	}
	if !strings.Contains(listOut, "Err Feed") {
		t.Fatalf("err feed should remain after dry-run: %s", listOut)
	}
}
