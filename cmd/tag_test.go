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

func TestTagsCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("tag", "1", "tech")

	out, err := executeCommand("tags", "--json")
	if err != nil {
		t.Fatalf("tags --json failed: %v", err)
	}
	if !strings.Contains(out, `"Name"`) || !strings.Contains(out, "tech") {
		t.Errorf("expected JSON output with tag data: %s", out)
	}
}

func TestTagsCmdYAML(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("tag", "1", "tech")

	out, err := executeCommand("tags", "--yaml")
	if err != nil {
		t.Fatalf("tags --yaml failed: %v", err)
	}
	if !strings.Contains(out, "name:") || !strings.Contains(out, "tech") {
		t.Errorf("expected YAML output with tag data: %s", out)
	}
}
