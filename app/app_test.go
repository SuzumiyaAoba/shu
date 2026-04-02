package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

func TestOpenInMemory(t *testing.T) {
	instance, err := Open(Config{
		DBPath:    ":memory:",
		LogLevel:  "info",
		LogOutput: new(bytes.Buffer),
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	if instance.Service == nil {
		t.Fatal("expected Service to be initialized")
	}
	if instance.Close == nil {
		t.Fatal("expected Close to be initialized")
	}
}

func TestOpenInvalidLogLevel(t *testing.T) {
	_, err := Open(Config{
		DBPath:    ":memory:",
		LogLevel:  "trace",
		LogOutput: new(bytes.Buffer),
	})
	if err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestOpenWithInjectedDependencies(t *testing.T) {
	testStore, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	t.Cleanup(func() { _ = testStore.Close() })

	logBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logBuffer, nil))

	var gotUserAgent string
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			gotUserAgent = req.Header.Get("User-Agent")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Injected</title><link>https://example.com</link><item><title>Post</title><link>https://example.com/post</link><guid>1</guid></item></channel></rss>`)),
			}, nil
		}),
	}

	instance, err := Open(Config{
		Logger:     logger,
		Store:      testStore,
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if instance.Store != testStore {
		t.Fatal("expected injected store to be reused")
	}
	if instance.Logger != logger {
		t.Fatal("expected injected logger to be reused")
	}

	if _, err := instance.Service.AddFeed(context.Background(), "https://example.com/feed.xml", ""); err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}
	if gotUserAgent != "shu/0.1" {
		t.Fatalf("User-Agent = %q, want %q", gotUserAgent, "shu/0.1")
	}
}

func TestOpenWithStoreOpenerAndCleanup(t *testing.T) {
	var openerCalled bool
	var cleanupCalled bool

	instance, err := Open(Config{
		DBPath:   ":memory:",
		LogLevel: "info",
		OpenStore: func(dsn string) (core.Store, error) {
			openerCalled = true
			s, err := store.NewSQLiteStore(dsn)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
		Cleanup: func() error {
			cleanupCalled = true
			return nil
		},
		LogOutput: new(bytes.Buffer),
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if !openerCalled {
		t.Fatal("expected OpenStore to be called")
	}
	if err := instance.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if !cleanupCalled {
		t.Fatal("expected Cleanup to be called")
	}
}

func TestOpenRejectsConflictingStoreConfig(t *testing.T) {
	_, err := Open(Config{
		DBPath: ":memory:",
		Store:  newFakeStore(),
		OpenStore: func(dsn string) (core.Store, error) {
			return newFakeStore(), nil
		},
	})
	if err == nil {
		t.Fatal("expected Store/OpenStore conflict")
	}
}

func TestOpenRejectsConflictingLoggerConfig(t *testing.T) {
	_, err := Open(Config{
		DBPath:    ":memory:",
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		LogLevel:  "info",
		LogOutput: new(bytes.Buffer),
		Store:     newFakeStore(),
	})
	if err == nil {
		t.Fatal("expected Logger conflict")
	}
}

func TestOpenRequiresDBPathWithoutStore(t *testing.T) {
	_, err := Open(Config{LogLevel: "info"})
	if err == nil {
		t.Fatal("expected missing DBPath error")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type fakeStore struct{}

func newFakeStore() *fakeStore { return &fakeStore{} }

func (f *fakeStore) AddFeed(context.Context, *core.Feed) error                           { return nil }
func (f *fakeStore) GetFeed(context.Context, int64) (*core.Feed, error)                  { return nil, nil }
func (f *fakeStore) GetFeedByURL(context.Context, string) (*core.Feed, error)            { return nil, nil }
func (f *fakeStore) ListFeeds(context.Context) ([]*core.Feed, error)                     { return nil, nil }
func (f *fakeStore) RemoveFeed(context.Context, int64) error                             { return nil }
func (f *fakeStore) UpdateFeed(context.Context, int64, core.FeedUpdate) error            { return nil }
func (f *fakeStore) UpdateFeedFetchedAt(context.Context, int64) error                    { return nil }
func (f *fakeStore) UpdateFeedCacheHeaders(context.Context, int64, string, string) error { return nil }
func (f *fakeStore) RecordFeedError(context.Context, int64, string) error                { return nil }
func (f *fakeStore) ResetFeedError(context.Context, int64) error                         { return nil }
func (f *fakeStore) SetFeedDisabled(context.Context, int64, bool) error                  { return nil }
func (f *fakeStore) AddEntries(context.Context, []*core.Entry) (int, error)              { return 0, nil }
func (f *fakeStore) GetEntry(context.Context, int64) (*core.Entry, error)                { return nil, nil }
func (f *fakeStore) ListEntries(context.Context, core.EntryFilter) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) CountEntries(context.Context, core.EntryFilter) (int, error) { return 0, nil }
func (f *fakeStore) SearchEntries(context.Context, string, int) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) FindDuplicateEntries(context.Context, int64) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) MarkEntryRead(context.Context, int64) error                   { return nil }
func (f *fakeStore) MarkEntriesRead(context.Context, []int64) error               { return nil }
func (f *fakeStore) MarkEntryUnread(context.Context, int64) error                 { return nil }
func (f *fakeStore) MarkEntriesUnread(context.Context, []int64) error             { return nil }
func (f *fakeStore) StarEntry(context.Context, int64) error                       { return nil }
func (f *fakeStore) StarEntries(context.Context, []int64) error                   { return nil }
func (f *fakeStore) UnstarEntry(context.Context, int64) error                     { return nil }
func (f *fakeStore) UnstarEntries(context.Context, []int64) error                 { return nil }
func (f *fakeStore) AddTag(context.Context, int64, string) error                  { return nil }
func (f *fakeStore) RemoveTag(context.Context, int64, string) error               { return nil }
func (f *fakeStore) ListTags(context.Context, int64) ([]core.Tag, error)          { return nil, nil }
func (f *fakeStore) ListFeedTags(context.Context) (map[int64][]core.Tag, error)   { return nil, nil }
func (f *fakeStore) ListAllTags(context.Context) ([]core.Tag, error)              { return nil, nil }
func (f *fakeStore) ListFeedsByTag(context.Context, string) ([]*core.Feed, error) { return nil, nil }
func (f *fakeStore) FeedStats(context.Context) ([]core.FeedStats, error)          { return nil, nil }
func (f *fakeStore) CleanupEntries(context.Context, time.Time) (int, error)       { return 0, nil }
func (f *fakeStore) Close() error                                                 { return nil }
