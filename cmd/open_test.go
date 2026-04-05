package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func TestOpenCmd(t *testing.T) {
	service := &openTestService{
		entry: &core.Entry{ID: 1, Link: "https://example.com/post-1"},
	}

	var openedURL string
	stdout, stderr, err := executeSingleCommand(newOpenCmd(func() (entryService, error) {
		return service, nil
	}, func(url string) error {
		openedURL = url
		return nil
	}), "1")
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}
	if openedURL != "https://example.com/post-1" {
		t.Fatalf("opened URL = %q", openedURL)
	}
	if !strings.Contains(stdout, "Opening https://example.com/post-1") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !service.markedRead {
		t.Fatal("expected entry to be marked as read")
	}
}

func TestOpenCmdWarnsWhenMarkReadFails(t *testing.T) {
	service := &openTestService{
		entry:       &core.Entry{ID: 1, Link: "https://example.com/post-1"},
		markReadErr: errors.New("boom"),
	}

	stdout, stderr, err := executeSingleCommand(newOpenCmd(func() (entryService, error) {
		return service, nil
	}, func(url string) error { return nil }), "1")
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}
	if !strings.Contains(stdout, "Opening https://example.com/post-1") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.Contains(stderr, "warning: could not mark entry as read: boom") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestOpenCmdFailsWhenBrowserOpenFails(t *testing.T) {
	service := &openTestService{
		entry: &core.Entry{ID: 1, Link: "https://example.com/post-1"},
	}

	_, _, err := executeSingleCommand(newOpenCmd(func() (entryService, error) {
		return service, nil
	}, func(url string) error { return errors.New("open failed") }), "1")
	if err == nil || !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("expected browser open error, got %v", err)
	}
}

func TestOpenCmdFailsWithoutLink(t *testing.T) {
	service := &openTestService{
		entry: &core.Entry{ID: 1},
	}

	_, _, err := executeSingleCommand(newOpenCmd(func() (entryService, error) {
		return service, nil
	}, func(url string) error { return nil }), "1")
	if err == nil || !strings.Contains(err.Error(), "entry #1 has no link") {
		t.Fatalf("expected missing link error, got %v", err)
	}
}

func TestOpenCmdWrapsGetEntryError(t *testing.T) {
	service := &openTestService{
		getEntryErr: errors.New("boom"),
	}

	_, _, err := executeSingleCommand(newOpenCmd(func() (entryService, error) {
		return service, nil
	}, func(url string) error { return nil }), "1")
	if err == nil || !strings.Contains(err.Error(), "entry #1: boom") {
		t.Fatalf("expected wrapped get entry error, got %v", err)
	}
}

func executeSingleCommand(cmd *cobra.Command, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

type openTestService struct {
	entry       *core.Entry
	getEntryErr error
	markReadErr error
	markedRead  bool
}

func (s *openTestService) GetEntry(context.Context, int64) (*core.Entry, error) {
	if s.getEntryErr != nil {
		return nil, s.getEntryErr
	}
	return s.entry, nil
}

func (s *openTestService) MarkEntryRead(context.Context, int64) error {
	s.markedRead = true
	return s.markReadErr
}

func (s *openTestService) ListEntries(context.Context, core.EntryFilter) ([]*core.Entry, error) {
	return nil, nil
}

func (s *openTestService) ListEntriesPage(context.Context, core.EntryFilter) (*core.EntryPage, error) {
	return nil, nil
}

func (s *openTestService) SearchEntries(context.Context, string, int) ([]*core.Entry, error) {
	return nil, nil
}

func (s *openTestService) SearchEntriesPage(context.Context, string, int, int) (*core.EntryPage, error) {
	return nil, nil
}

func (s *openTestService) FindDuplicateEntries(context.Context, int64) ([]*core.Entry, error) {
	return nil, nil
}

func (s *openTestService) MarkEntriesRead(context.Context, []int64) error { return nil }

func (s *openTestService) MarkEntryUnread(context.Context, int64) error { return nil }

func (s *openTestService) MarkEntriesUnread(context.Context, []int64) error { return nil }

func (s *openTestService) StarEntry(context.Context, int64) error { return nil }

func (s *openTestService) StarEntries(context.Context, []int64) error { return nil }

func (s *openTestService) UnstarEntry(context.Context, int64) error { return nil }

func (s *openTestService) UnstarEntries(context.Context, []int64) error { return nil }

func TestOpenBrowserRejectsNonHTTP(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com", false},
		{"http://example.com", false},
		{"javascript:alert(1)", true},
		{"file:///etc/passwd", true},
		{"ftp://example.com/feed.xml", true},
		{"", true},
		{"://bad", true},
	}
	for _, tt := range tests {
		err := openBrowser(tt.url)
		if tt.wantErr && err == nil {
			t.Errorf("openBrowser(%q): expected error, got nil", tt.url)
		}
		if !tt.wantErr && err != nil {
			// On CI the "open" command may not exist, so we only check
			// that the scheme validation itself did not reject the URL.
			if strings.Contains(err.Error(), "refusing to open") {
				t.Errorf("openBrowser(%q): unexpected scheme rejection: %v", tt.url, err)
			}
		}
	}
}
