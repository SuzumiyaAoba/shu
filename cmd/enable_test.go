package cmd

import (
	"strings"
	"testing"
)

func TestEnableDisableCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("disable", "1")
	if err != nil {
		t.Fatalf("disable command failed: %v", err)
	}
	if !strings.Contains(out, "Disabled feed #1") {
		t.Errorf("unexpected output: %s", out)
	}

	// List should show disabled status.
	listOut, _ := executeCommand("list")
	if !strings.Contains(listOut, "disabled") {
		t.Errorf("expected disabled in list: %s", listOut)
	}

	out, err = executeCommand("enable", "1")
	if err != nil {
		t.Fatalf("enable command failed: %v", err)
	}
	if !strings.Contains(out, "Enabled feed #1") {
		t.Errorf("unexpected output: %s", out)
	}
}
