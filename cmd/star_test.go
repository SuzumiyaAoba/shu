package cmd

import (
	"strings"
	"testing"
)

func TestStarUnstarCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	out, err := executeCommand("star", "1")
	if err != nil {
		t.Fatalf("star command failed: %v", err)
	}
	if !strings.Contains(out, "Starred entry #1") {
		t.Errorf("unexpected output: %s", out)
	}

	// Starred entries filter.
	starredOut, _ := executeCommand("entries", "--starred")
	if !strings.Contains(starredOut, "Post") {
		t.Errorf("expected starred entry in output: %s", starredOut)
	}

	out, err = executeCommand("unstar", "1")
	if err != nil {
		t.Fatalf("unstar command failed: %v", err)
	}
	if !strings.Contains(out, "Unstarred entry #1") {
		t.Errorf("unexpected output: %s", out)
	}
}
