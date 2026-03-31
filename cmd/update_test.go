package cmd

import (
	"strings"
	"testing"
)

func TestUpdateCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("update", "1", "--title", "New Title")
	if err != nil {
		t.Fatalf("update command failed: %v", err)
	}
	if !strings.Contains(out, "Updated feed #1") {
		t.Errorf("unexpected output: %s", out)
	}

	listOut, _ := executeCommand("list")
	if !strings.Contains(listOut, "New Title") {
		t.Errorf("expected new title in list: %s", listOut)
	}
}
