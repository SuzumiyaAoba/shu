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

func TestSearchCmdPageInfoJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("search", "Post", "--limit", "1", "--page", "1", "--page-info", "--json")
	if err != nil {
		t.Fatalf("search --page-info --json failed: %v", err)
	}
	if !strings.Contains(out, `"total_count"`) || !strings.Contains(out, `"has_more"`) {
		t.Errorf("expected page metadata in output: %s", out)
	}
}

func TestSearchCmdPageInfoTable(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("search", "Post", "--limit", "1", "--page-info")
	if err != nil {
		t.Fatalf("search --page-info failed: %v", err)
	}
	if !strings.Contains(out, "Showing 1/2 results") {
		t.Errorf("expected page footer in output: %s", out)
	}
}

func TestSearchCmdYAML(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("search", "Post", "--yaml")
	if err != nil {
		t.Fatalf("search --yaml failed: %v", err)
	}
	if !strings.Contains(out, "title: Post 1") {
		t.Errorf("expected YAML output: %s", out)
	}
}
