package cmd

import (
	"strings"
	"testing"
)

func TestReadUnreadCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")
	_, _ = executeCommand("fetch")

	// Mark as read.
	out, err := executeCommand("read", "1")
	if err != nil {
		t.Fatalf("read command failed: %v", err)
	}
	if !strings.Contains(out, "Marked entry #1 as read") {
		t.Errorf("unexpected output: %s", out)
	}

	// Unread entries should exclude it.
	unreadOut, _ := executeCommand("entries", "--unread")
	if strings.Contains(unreadOut, "\n1\t") {
		t.Errorf("entry 1 should not appear in unread list")
	}

	// Mark as unread.
	out, err = executeCommand("unread", "1")
	if err != nil {
		t.Fatalf("unread command failed: %v", err)
	}
	if !strings.Contains(out, "Marked entry #1 as unread") {
		t.Errorf("unexpected output: %s", out)
	}
}
