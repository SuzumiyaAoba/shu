package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

var testService *core.Service

// setupTest creates an in-memory Service backed by a real SQLite store and a
// local httptest server serving testRSSFeed, then registers that service for
// executeCommand.
func setupTest(t *testing.T) string {
	t.Helper()

	service, ts := newCommandTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, testRSSFeed)
	}))
	setTestService(t, service)

	return ts.URL
}

func newCommandTestService(t *testing.T, handler http.Handler, options ...core.Option) (*core.Service, *httptest.Server) {
	t.Helper()

	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var ts *httptest.Server
	if handler != nil {
		ts = httptest.NewServer(handler)
		t.Cleanup(ts.Close)
		options = append([]core.Option{core.WithHTTPClient(ts.Client())}, options...)
	}

	return core.New(s, logger, options...), ts
}

func setTestService(t *testing.T, service *core.Service) {
	t.Helper()
	testService = service
	t.Cleanup(func() {
		testService = nil
	})
}

// executeCommand runs a Cobra command with the given arguments and returns the
// combined stdout output and any error. It replaces the command's stdout with a
// buffer to capture output.
func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd := newRootCmd(testService)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func writeTempFile(t *testing.T, pattern, content string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	if _, err := io.WriteString(f, content); err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	return f.Name()
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
