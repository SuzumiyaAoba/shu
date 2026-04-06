// Package fetch implements the feed download, parse, and persistence pipeline.
package fetch

import "sync"

// EventType identifies the phase of a fetch lifecycle notification.
type EventType string

const (
	// EventStarted is emitted when a feed fetch begins.
	EventStarted EventType = "started"
	// EventSkipped is emitted when a feed is skipped without fetching new content.
	EventSkipped EventType = "skipped"
	// EventCompleted is emitted when a feed fetch finishes, with or without an error.
	EventCompleted EventType = "completed"
)

// SkipReason explains why a feed did not produce a fetch result.
type SkipReason string

const (
	// SkipDisabled indicates the feed is currently disabled.
	SkipDisabled SkipReason = "disabled"
	// SkipInterval indicates the feed was skipped because its per-feed interval has not elapsed.
	SkipInterval SkipReason = "interval"
	// SkipNotModified indicates the server returned HTTP 304 Not Modified.
	SkipNotModified SkipReason = "not_modified"
)

// Event is a structured notification about feed fetch progress.
type Event struct {
	Type       EventType
	FeedID     int64
	FeedTitle  string
	FeedURL    string
	NewEntries int
	SkipReason SkipReason
	Err        error
}

// Observer receives feed fetch lifecycle notifications.
// Callback invocations are serialized by the service, even when feeds are
// fetched concurrently.
type Observer interface {
	OnFetchEvent(event Event)
}

// ObserverFunc adapts a function to [Observer].
type ObserverFunc func(event Event)

// OnFetchEvent calls fn(event).
func (fn ObserverFunc) OnFetchEvent(event Event) {
	fn(event)
}

type notifier struct {
	observer Observer
	mu       sync.Mutex
}

func newNotifier(observer Observer) *notifier {
	return &notifier{observer: observer}
}

func (n *notifier) emit(event Event) {
	if n == nil || n.observer == nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.observer.OnFetchEvent(event)
}

func (n *notifier) started(feedID int64, feedTitle, feedURL string) {
	n.emit(Event{
		Type:      EventStarted,
		FeedID:    feedID,
		FeedTitle: feedTitle,
		FeedURL:   feedURL,
	})
}

func (n *notifier) skipped(feedID int64, feedTitle, feedURL string, reason SkipReason) {
	n.emit(Event{
		Type:       EventSkipped,
		FeedID:     feedID,
		FeedTitle:  feedTitle,
		FeedURL:    feedURL,
		SkipReason: reason,
	})
}

func (n *notifier) completed(feedID int64, feedTitle, feedURL string, newEntries int, err error) {
	n.emit(Event{
		Type:       EventCompleted,
		FeedID:     feedID,
		FeedTitle:  feedTitle,
		FeedURL:    feedURL,
		NewEntries: newEntries,
		Err:        err,
	})
}
