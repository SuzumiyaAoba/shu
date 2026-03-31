package cmd

import (
	"strings"
	"testing"
)

func TestEntriesCmdEmpty(t *testing.T) {
	setupTest(t)

	out, err := executeCommand("entries")
	if err != nil {
		t.Fatalf("entries command failed: %v", err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "TITLE") {
		t.Errorf("expected table header in output: %s", out)
	}
}

func TestEntriesCmdWithData(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("entries")
	if err != nil {
		t.Fatalf("entries command failed: %v", err)
	}
	if !strings.Contains(out, "Post 1") || !strings.Contains(out, "Post 2") {
		t.Errorf("expected entry titles in output: %s", out)
	}
}

func TestEntriesCmdFilterByFeed(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("entries", "--feed-id", "1")
	if err != nil {
		t.Fatalf("entries --feed-id failed: %v", err)
	}
	if !strings.Contains(out, "Post 1") {
		t.Errorf("expected entries in output: %s", out)
	}
}

func TestEntriesCmdLimit(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("entries", "--limit", "1")
	if err != nil {
		t.Fatalf("entries --limit failed: %v", err)
	}
	// Should contain header + 1 data row.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + 1 entry), got %d: %s", len(lines), out)
	}
}

func TestEntriesCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("entries", "--json")
	if err != nil {
		t.Fatalf("entries --json failed: %v", err)
	}
	if !strings.Contains(out, `"Title"`) || !strings.Contains(out, "Post 1") {
		t.Errorf("expected JSON output: %s", out)
	}
}
