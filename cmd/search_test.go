package cmd

import (
	"strings"
	"testing"
)

func TestSearchCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("search", "Post")
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
	if !strings.Contains(out, "Post 1") {
		t.Errorf("expected Post 1 in search results: %s", out)
	}
}

func TestSearchCmdNoResults(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("search", "nonexistent_term_xyz")
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
	// Should just have the header.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only header, got %d lines: %s", len(lines), out)
	}
}
