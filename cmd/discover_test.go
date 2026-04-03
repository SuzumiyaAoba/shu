package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
)

func TestDiscoverCmd(t *testing.T) {
	htmlPage := `<!DOCTYPE html>
<html><head>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head><body></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, htmlPage)
	}))
	defer ts.Close()

	service, _ := newCommandTestService(t, nil, core.WithHTTPClient(ts.Client()))
	setTestService(t, service)

	out, err := executeCommand("discover", ts.URL)
	if err != nil {
		t.Fatalf("discover command failed: %v", err)
	}
	if !strings.Contains(out, "/feed.xml") {
		t.Errorf("expected feed URL in output: %s", out)
	}
}

func TestDiscoverCmdJSON(t *testing.T) {
	htmlPage := `<!DOCTYPE html>
<html><head>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head><body></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, htmlPage)
	}))
	defer ts.Close()

	service, _ := newCommandTestService(t, nil, core.WithHTTPClient(ts.Client()))
	setTestService(t, service)

	out, err := executeCommand("discover", ts.URL, "--json")
	if err != nil {
		t.Fatalf("discover --json failed: %v", err)
	}
	if !strings.Contains(out, "/feed.xml") || !strings.Contains(out, "[") {
		t.Errorf("expected JSON array with feed URL: %s", out)
	}
}

func TestDiscoverCmdYAML(t *testing.T) {
	htmlPage := `<!DOCTYPE html>
<html><head>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head><body></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, htmlPage)
	}))
	defer ts.Close()

	service, _ := newCommandTestService(t, nil, core.WithHTTPClient(ts.Client()))
	setTestService(t, service)

	out, err := executeCommand("discover", ts.URL, "--yaml")
	if err != nil {
		t.Fatalf("discover --yaml failed: %v", err)
	}
	if !strings.Contains(out, "/feed.xml") {
		t.Errorf("expected YAML output with feed URL: %s", out)
	}
}
