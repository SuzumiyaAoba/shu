package cmd

import (
	"strings"
	"testing"
)

func TestAddCmd(t *testing.T) {
	tsURL := setupTest(t)

	out, err := executeCommand("add", tsURL+"/feed.xml")
	if err != nil {
		t.Fatalf("add command failed: %v", err)
	}
	if !strings.Contains(out, "Added feed #") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "Test Blog") {
		t.Errorf("expected title in output: %s", out)
	}
}

func TestAddCmdWithTitle(t *testing.T) {
	tsURL := setupTest(t)

	out, err := executeCommand("add", tsURL+"/feed.xml", "--title", "Custom Title")
	if err != nil {
		t.Fatalf("add command failed: %v", err)
	}
	if !strings.Contains(out, "Custom Title") {
		t.Errorf("expected custom title in output: %s", out)
	}
}

func TestAddCmdMissingArg(t *testing.T) {
	setupTest(t)

	_, err := executeCommand("add")
	if err == nil {
		t.Error("expected error for missing URL argument")
	}
}

func TestAddCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	out, err := executeCommand("add", tsURL+"/feed.xml", "--json")
	if err != nil {
		t.Fatalf("add --json failed: %v", err)
	}
	if !strings.Contains(out, `"title"`) || !strings.Contains(out, "Test Blog") {
		t.Errorf("expected JSON output with feed data: %s", out)
	}
}

func TestAddCmdYAML(t *testing.T) {
	tsURL := setupTest(t)

	out, err := executeCommand("add", tsURL+"/feed.xml", "--yaml")
	if err != nil {
		t.Fatalf("add --yaml failed: %v", err)
	}
	if !strings.Contains(out, "title:") || !strings.Contains(out, "Test Blog") {
		t.Errorf("expected YAML output with feed data: %s", out)
	}
}
