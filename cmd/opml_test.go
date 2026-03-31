package cmd

import (
	"strings"
	"testing"
)

func TestExportCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("export")
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
	if !strings.Contains(out, "<opml") {
		t.Errorf("expected OPML output: %s", out)
	}
	if !strings.Contains(out, "xmlUrl=") {
		t.Errorf("expected feed URL in OPML: %s", out)
	}
}
