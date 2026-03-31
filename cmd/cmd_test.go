package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

// setupTest creates an in-memory Service backed by a real SQLite store and a
// local httptest server serving testRSSFeed. It injects the service into the
// package-level svc variable used by all subcommands, disables the root
// command's PersistentPreRunE/PostRunE hooks (which would overwrite svc with a
// file-backed store), and returns the test server URL.
func setupTest(t *testing.T) string {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		io.WriteString(w, testRSSFeed)
	}))
	t.Cleanup(ts.Close)

	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.New(s, logger)
	service.SetHTTPClient(ts.Client())

	svc = service

	// Disable the lifecycle hooks so PersistentPreRunE doesn't overwrite svc
	// with a file-backed store and PersistentPostRunE doesn't close it.
	origPre := rootCmd.PersistentPreRunE
	origPost := rootCmd.PersistentPostRunE
	rootCmd.PersistentPreRunE = nil
	rootCmd.PersistentPostRunE = nil
	t.Cleanup(func() {
		rootCmd.PersistentPreRunE = origPre
		rootCmd.PersistentPostRunE = origPost
	})

	return ts.URL
}

// executeCommand runs a Cobra command with the given arguments and returns the
// combined stdout output and any error. It replaces the command's stdout with a
// buffer to capture output.
func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset flag values that persist between test runs due to package-level vars.
	resetFlags()

	err := rootCmd.Execute()
	return buf.String(), err
}

// resetFlags resets subcommand flag values to their defaults so that flag state
// from one test does not leak into the next.
func resetFlags() {
	addTitle = ""
	addJSON = false
	listJSON = false
	fetchFeedID = 0
	fetchJSON = false
	entriesFeedID = 0
	entriesLimit = 20
	entriesJSON = false
	entriesUnread = false
	entriesStarred = false
	entriesTag = ""
	entriesFormat = ""
	statsJSON = false
	cleanupOlderThan = 90 * 24 * time.Hour
	updateTitle = ""
	updateURL = ""
	searchLimit = 20
	searchJSON = false
	discoverJSON = false
	duplicatesJSON = false
	tagsJSON = false
}

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Blog</title>
    <link>https://example.com</link>
    <description>A blog about testing</description>
    <item>
      <title>Post 1</title>
      <link>https://example.com/post-1</link>
      <guid>post-1</guid>
      <description>First post</description>
    </item>
    <item>
      <title>Post 2</title>
      <link>https://example.com/post-2</link>
      <guid>post-2</guid>
      <description>Second post</description>
    </item>
  </channel>
</rss>`
