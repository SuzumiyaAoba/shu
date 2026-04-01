package cmd

import (
	"strings"
	"testing"
)

func TestRootHelpShowsCommandGroups(t *testing.T) {
	setupTest(t)

	out, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	for _, group := range []string{
		"Feed Commands",
		"Entry Commands",
		"Tag Commands",
		"Maintenance Commands",
		"Import/Export Commands",
	} {
		if !strings.Contains(out, group) {
			t.Fatalf("expected help to contain %q, got:\n%s", group, out)
		}
	}
}
