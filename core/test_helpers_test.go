package core_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

func newTestService(t *testing.T, handler http.Handler) *core.Service {
	return newTestServiceWithOptions(t, handler)
}

func newTestServiceWithOptions(t *testing.T, handler http.Handler, options ...core.Option) *core.Service {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if handler != nil {
		ts := httptest.NewServer(handler)
		t.Cleanup(ts.Close)
		options = append([]core.Option{core.WithHTTPClient(ts.Client())}, options...)
	}

	return core.New(s, logger, options...)
}

func newStaticTestServer(t *testing.T, contentType, body string) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func newFeedTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newStaticTestServer(t, "application/rss+xml", testRSSFeed)
}

func newHTMLTestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return newStaticTestServer(t, "text/html", body)
}

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test Blog</title>
    <link>https://example.com</link>
    <description>A blog about testing</description>
    <language>en</language>
    <image>
      <url>https://example.com/logo.png</url>
      <title>Test Blog</title>
    </image>
    <item>
      <title>Post 1</title>
      <link>https://example.com/post-1</link>
      <guid>post-1</guid>
      <description>First post</description>
      <content:encoded><![CDATA[<p>Full content of post 1</p>]]></content:encoded>
      <author>alice@example.com (Alice)</author>
      <category>Go</category>
      <category>Testing</category>
      <enclosure url="https://example.com/ep1.mp3" length="12345" type="audio/mpeg"/>
    </item>
    <item>
      <title>Post 2</title>
      <link>https://example.com/post-2</link>
      <guid>post-2</guid>
      <description>Second post</description>
    </item>
  </channel>
</rss>`
