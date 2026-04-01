package core

import "sync"

// FetchEventType identifies the phase of a fetch lifecycle notification.
type FetchEventType string

const (
	// FetchEventStarted is emitted when a feed fetch begins.
	FetchEventStarted FetchEventType = "started"
	// FetchEventSkipped is emitted when a feed is skipped without fetching new content.
	FetchEventSkipped FetchEventType = "skipped"
	// FetchEventCompleted is emitted when a feed fetch finishes, with or without an error.
	FetchEventCompleted FetchEventType = "completed"
)

// FetchSkipReason explains why a feed did not produce a fetch result.
type FetchSkipReason string

const (
	// FetchSkipDisabled indicates the feed is currently disabled.
	FetchSkipDisabled FetchSkipReason = "disabled"
	// FetchSkipInterval indicates the feed was skipped because its per-feed interval has not elapsed.
	FetchSkipInterval FetchSkipReason = "interval"
	// FetchSkipNotModified indicates the server returned HTTP 304 Not Modified.
	FetchSkipNotModified FetchSkipReason = "not_modified"
)

// FetchEvent is a structured notification about feed fetch progress.
type FetchEvent struct {
	Type       FetchEventType
	FeedID     int64
	FeedTitle  string
	FeedURL    string
	NewEntries int
	SkipReason FetchSkipReason
	Err        error
}

// FetchObserver receives feed fetch lifecycle notifications.
// Callback invocations are serialized by the service, even when feeds are
// fetched concurrently.
type FetchObserver interface {
	OnFetchEvent(event FetchEvent)
}

// FetchObserverFunc adapts a function to [FetchObserver].
type FetchObserverFunc func(event FetchEvent)

// OnFetchEvent calls fn(event).
func (fn FetchObserverFunc) OnFetchEvent(event FetchEvent) {
	fn(event)
}

type fetchNotifier struct {
	observer FetchObserver
	mu       sync.Mutex
}

func newFetchNotifier(observer FetchObserver) *fetchNotifier {
	return &fetchNotifier{observer: observer}
}

func (n *fetchNotifier) emit(event FetchEvent) {
	if n == nil || n.observer == nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.observer.OnFetchEvent(event)
}
