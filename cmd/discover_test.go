package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SuzumiyaAoba/shu/store"
	"github.com/SuzumiyaAoba/shu/core"
	"log/slog"
)

func TestDiscoverCmd(t *testing.T) {
	htmlPage := `<!DOCTYPE html>
<html><head>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head><body></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, htmlPage)
	}))
	defer ts.Close()

	// Need a custom setup since discover uses a different server response.
	s, _ := store.NewSQLiteStore(":memory:")
	t.Cleanup(func() { s.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.New(s, logger)
	service.SetHTTPClient(ts.Client())
	svc = service

	origPre := rootCmd.PersistentPreRunE
	origPost := rootCmd.PersistentPostRunE
	rootCmd.PersistentPreRunE = nil
	rootCmd.PersistentPostRunE = nil
	t.Cleanup(func() {
		rootCmd.PersistentPreRunE = origPre
		rootCmd.PersistentPostRunE = origPost
	})

	out, err := executeCommand("discover", ts.URL)
	if err != nil {
		t.Fatalf("discover command failed: %v", err)
	}
	if !strings.Contains(out, "/feed.xml") {
		t.Errorf("expected feed URL in output: %s", out)
	}
}
