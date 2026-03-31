package cmd

import (
	"strings"
	"testing"
)

func TestTagUntagCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	// Tag.
	out, err := executeCommand("tag", "1", "tech")
	if err != nil {
		t.Fatalf("tag command failed: %v", err)
	}
	if !strings.Contains(out, `Tagged feed #1 with "tech"`) {
		t.Errorf("unexpected output: %s", out)
	}

	// List tags for feed.
	tagsOut, err := executeCommand("tags", "1")
	if err != nil {
		t.Fatalf("tags command failed: %v", err)
	}
	if !strings.Contains(tagsOut, "tech") {
		t.Errorf("expected tech in tags: %s", tagsOut)
	}

	// List all tags.
	allTagsOut, err := executeCommand("tags")
	if err != nil {
		t.Fatalf("tags command failed: %v", err)
	}
	if !strings.Contains(allTagsOut, "tech") {
		t.Errorf("expected tech in all tags: %s", allTagsOut)
	}

	// Untag.
	out, err = executeCommand("untag", "1", "tech")
	if err != nil {
		t.Fatalf("untag command failed: %v", err)
	}
	if !strings.Contains(out, `Removed tag "tech" from feed #1`) {
		t.Errorf("unexpected output: %s", out)
	}
}
