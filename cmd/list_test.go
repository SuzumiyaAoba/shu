package cmd

import (
	"strings"
	"testing"
)

func TestListCmdEmpty(t *testing.T) {
	setupTest(t)

	out, err := executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "TITLE") {
		t.Errorf("expected table header in output: %s", out)
	}
}

func TestListCmdWithFeeds(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	if !strings.Contains(out, "Test Blog") {
		t.Errorf("expected feed title in output: %s", out)
	}
}

func TestListCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("list", "--json")
	if err != nil {
		t.Fatalf("list --json failed: %v", err)
	}
	if !strings.Contains(out, `"Title"`) || !strings.Contains(out, "Test Blog") {
		t.Errorf("expected JSON output with feed data: %s", out)
	}
}

func TestListCmdYAML(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("list", "--yaml")
	if err != nil {
		t.Fatalf("list --yaml failed: %v", err)
	}
	if !strings.Contains(out, "title:") || !strings.Contains(out, "Test Blog") {
		t.Errorf("expected YAML output with feed data: %s", out)
	}
}
