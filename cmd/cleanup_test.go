package cmd

import (
	"strings"
	"testing"
)

func TestCleanupCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("cleanup", "--older-than", "0s")
	if err != nil {
		t.Fatalf("cleanup command failed: %v", err)
	}
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected deletion message: %s", out)
	}
}
