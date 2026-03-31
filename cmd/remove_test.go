package cmd

import (
	"strings"
	"testing"
)

func TestRemoveCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("remove", "1")
	if err != nil {
		t.Fatalf("remove command failed: %v", err)
	}
	if !strings.Contains(out, "Removed feed #1") {
		t.Errorf("unexpected output: %s", out)
	}

	// Verify the feed is gone from the list.
	listOut, _ := executeCommand("list")
	if strings.Contains(listOut, "Test Blog") {
		t.Error("feed should have been removed from list")
	}
}

func TestRemoveCmdInvalidID(t *testing.T) {
	setupTest(t)

	_, err := executeCommand("remove", "abc")
	if err == nil {
		t.Error("expected error for non-numeric ID")
	}
}

func TestRemoveCmdMissingArg(t *testing.T) {
	setupTest(t)

	_, err := executeCommand("remove")
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}
