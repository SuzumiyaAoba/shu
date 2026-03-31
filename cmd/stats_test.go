package cmd

import (
	"strings"
	"testing"
)

func TestStatsCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("stats")
	if err != nil {
		t.Fatalf("stats command failed: %v", err)
	}
	if !strings.Contains(out, "TOTAL") || !strings.Contains(out, "UNREAD") {
		t.Errorf("expected stats header: %s", out)
	}
	if !strings.Contains(out, "Test Blog") {
		t.Errorf("expected feed title in stats: %s", out)
	}
}
