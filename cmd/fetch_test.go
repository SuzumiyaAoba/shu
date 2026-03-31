package cmd

import (
	"strings"
	"testing"
)

func TestFetchCmdAll(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("fetch")
	if err != nil {
		t.Fatalf("fetch command failed: %v", err)
	}
	if !strings.Contains(out, "Fetched") || !strings.Contains(out, "new entries") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestFetchCmdByFeedID(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("fetch", "--feed-id", "1")
	if err != nil {
		t.Fatalf("fetch --feed-id failed: %v", err)
	}
	if !strings.Contains(out, "feed #1") {
		t.Errorf("expected feed ID in output: %s", out)
	}
}

func TestFetchCmdNoFeeds(t *testing.T) {
	setupTest(t)

	out, err := executeCommand("fetch")
	if err != nil {
		t.Fatalf("fetch command failed: %v", err)
	}
	if !strings.Contains(out, "Fetched 0 new entries") {
		t.Errorf("expected 0 entries for empty feed list: %s", out)
	}
}

func TestFetchCmdJSON(t *testing.T) {
	setupTest(t)

	out, err := executeCommand("fetch", "--json")
	if err != nil {
		t.Fatalf("fetch --json failed: %v", err)
	}
	if !strings.Contains(out, `"count"`) {
		t.Errorf("expected JSON output with count: %s", out)
	}
}

func TestFetchCmdByFeedIDJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("fetch", "--feed-id", "1", "--json")
	if err != nil {
		t.Fatalf("fetch --feed-id --json failed: %v", err)
	}
	// Should output a JSON array (possibly empty if entries were already fetched during add).
	if !strings.Contains(out, "[") {
		t.Errorf("expected JSON array in output: %s", out)
	}
}
