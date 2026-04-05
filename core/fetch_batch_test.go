package core

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestFetchFeedsConcurrentlyRecoversPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	panicFeed := &Feed{ID: 1, Title: "panic-feed", URL: "https://example.com/panic"}

	fetcher := &Fetcher{
		logger: logger,
		downloader: downloadFunc(func(_ context.Context, feed *Feed) (*fetchedFeedDocument, bool, error) {
			if feed.ID == panicFeed.ID {
				panic("test panic in worker")
			}
			return nil, true, nil
		}),
	}

	notifier := newFetchNotifier(nil)
	_, err := fetcher.fetchFeedsConcurrently(context.Background(), []*Feed{panicFeed}, notifier)
	if err == nil {
		t.Fatal("expected error from panicked worker, got nil")
	}
	if got := err.Error(); got != "worker panic: test panic in worker" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

// downloadFunc adapts a plain function to the feedDownloader interface.
type downloadFunc func(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error)

func (f downloadFunc) download(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error) {
	return f(ctx, feed)
}
