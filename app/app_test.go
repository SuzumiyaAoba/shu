package app

import (
	"bytes"
	"context"
	"errors"
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

func TestOpenRejectsConflictingSQLiteOptions(t *testing.T) {
	_, err := Open(Config{
		DBPath:        ":memory:",
		Store:         newFakeStore(),
		SQLiteOptions: &store.SQLiteOptions{BusyTimeout: time.Second},
	})
	if err == nil {
		t.Fatal("expected Store/SQLiteOptions conflict")
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

func TestOpenWithSQLiteOptions(t *testing.T) {
	instance, err := Open(Config{
		DBPath:   ":memory:",
		LogLevel: "info",
		SQLiteOptions: &store.SQLiteOptions{
			BusyTimeout:  2 * time.Second,
			MaxOpenConns: 2,
		},
		LogOutput: new(bytes.Buffer),
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = instance.Close() })
}

func TestOpenCleanupRunsOwnedCloseBeforeCleanup(t *testing.T) {
	var calls []string

	instance, err := Open(Config{
		DBPath:   ":memory:",
		LogLevel: "info",
		OpenStore: func(dsn string) (core.Store, error) {
			return &trackingStore{
				fakeStore: newFakeStore(),
				closeFn: func() error {
					calls = append(calls, "store")
					return nil
				},
			}, nil
		},
		Cleanup: func() error {
			calls = append(calls, "cleanup")
			return nil
		},
		LogOutput: new(bytes.Buffer),
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := instance.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if len(calls) != 2 || calls[0] != "store" || calls[1] != "cleanup" {
		t.Fatalf("cleanup order = %v, want [store cleanup]", calls)
	}
}

func TestOpenCleanupStopsAfterFirstError(t *testing.T) {
	cleanupCalled := false

	instance, err := Open(Config{
		DBPath:   ":memory:",
		LogLevel: "info",
		OpenStore: func(dsn string) (core.Store, error) {
			return &trackingStore{
				fakeStore: newFakeStore(),
				closeFn: func() error {
					return errors.New("close failed")
				},
			}, nil
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

	err = instance.Close()
	if err == nil || err.Error() != "close failed" {
		t.Fatalf("Close error = %v, want close failed", err)
	}
	if cleanupCalled {
		t.Fatal("expected cleanup to be skipped after close error")
	}
}
