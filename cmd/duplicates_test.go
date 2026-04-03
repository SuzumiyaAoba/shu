package cmd

import (
	"strings"
	"testing"
)

func TestDuplicatesCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed-a.xml")
	_, _ = executeCommand("add", tsURL+"/feed-b.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("duplicates", "1")
	if err != nil {
		t.Fatalf("duplicates command failed: %v", err)
	}
	if !strings.Contains(out, "Post 1") {
		t.Errorf("expected duplicate entry in output: %s", out)
	}
}

func TestDuplicatesCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed-a.xml")
	_, _ = executeCommand("add", tsURL+"/feed-b.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("duplicates", "1", "--json")
	if err != nil {
		t.Fatalf("duplicates --json failed: %v", err)
	}
	if !strings.Contains(out, `"title"`) || !strings.Contains(out, "Post 1") {
		t.Errorf("expected JSON duplicate output: %s", out)
	}
}

func TestDuplicatesCmdNoResults(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed-a.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("duplicates", "1")
	if err != nil {
		t.Fatalf("duplicates command failed: %v", err)
	}
	if !strings.Contains(out, "No duplicates found.") {
		t.Errorf("expected empty duplicate message: %s", out)
	}
}
