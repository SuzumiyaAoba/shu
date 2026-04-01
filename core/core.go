// Package core contains the business logic for the shu RSS aggregator.
//
// This package is intentionally free of infrastructure dependencies (no SQLite,
// no CLI framework). It defines its own [Store] interface and operates
// exclusively through that abstraction, allowing the same logic to be driven by
// a CLI, an AWS Lambda handler, or any other composition root.
package core

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// FeedStore handles feed CRUD operations.
type FeedStore interface {
	AddFeed(ctx context.Context, feed *Feed) error
	GetFeed(ctx context.Context, id int64) (*Feed, error)
	ListFeeds(ctx context.Context) ([]*Feed, error)
	RemoveFeed(ctx context.Context, id int64) error
	UpdateFeed(ctx context.Context, id int64, update FeedUpdate) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
}

// FeedHealthStore tracks feed fetch errors and disabled state.
type FeedHealthStore interface {
	RecordFeedError(ctx context.Context, id int64, errMsg string) error
	ResetFeedError(ctx context.Context, id int64) error
	SetFeedDisabled(ctx context.Context, id int64, disabled bool) error
}

// EntryStore handles entry persistence and queries.
type EntryStore interface {
	AddEntries(ctx context.Context, entries []*Entry) (int, error)
	GetEntry(ctx context.Context, id int64) (*Entry, error)
	ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error)
	SearchEntries(ctx context.Context, query string, limit int) ([]*Entry, error)
	FindDuplicateEntries(ctx context.Context, entryID int64) ([]*Entry, error)
}

// EntryStateStore manages read/star state on entries.
type EntryStateStore interface {
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntriesRead(ctx context.Context, ids []int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	MarkEntriesUnread(ctx context.Context, ids []int64) error
	StarEntry(ctx context.Context, id int64) error
	StarEntries(ctx context.Context, ids []int64) error
	UnstarEntry(ctx context.Context, id int64) error
	UnstarEntries(ctx context.Context, ids []int64) error
}

// TagStore handles tag CRUD and feed-tag associations.
type TagStore interface {
	AddTag(ctx context.Context, feedID int64, tagName string) error
	RemoveTag(ctx context.Context, feedID int64, tagName string) error
	ListTags(ctx context.Context, feedID int64) ([]Tag, error)
	ListFeedTags(ctx context.Context) (map[int64][]Tag, error)
	ListAllTags(ctx context.Context) ([]Tag, error)
	ListFeedsByTag(ctx context.Context, tagName string) ([]*Feed, error)
}

// MaintenanceStore provides housekeeping operations.
type MaintenanceStore interface {
	FeedStats(ctx context.Context) ([]FeedStats, error)
	CleanupEntries(ctx context.Context, olderThan time.Time) (int, error)
}

// Store is the full persistence contract required by [Service].
// It composes all role-specific interfaces. Implementations must be safe for
// concurrent use.
type Store interface {
	FeedStore
	FeedHealthStore
	EntryStore
	EntryStateStore
	TagStore
	MaintenanceStore
	Close() error
}

// userAgent is the User-Agent string sent with every outbound HTTP request made
// by the service. It identifies the client to feed servers.
const userAgent = "shu/0.1"

// userAgentTransport is an [http.RoundTripper] decorator that injects the shu
// User-Agent header into every outbound request before delegating to the
// underlying transport.
type userAgentTransport struct {
	base http.RoundTripper
}

// RoundTrip clones the request, sets the User-Agent header, and forwards it to
// the base transport.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent)
	return t.base.RoundTrip(req)
}

// Service is the central orchestrator for feed management and fetching.
// It coordinates between the HTTP client (for downloading feeds) and the
// [Store] (for persistence), and emits structured log messages for
// observability.
type Service struct {
	store    Store
	logger   *slog.Logger
	client   *http.Client
	clientMu sync.RWMutex
}

// New creates a [Service] with the given store and logger.
// The returned service uses an HTTP client with a 30-second timeout and a
// custom transport that sets the User-Agent header to "shu/0.1".
func New(store Store, logger *slog.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &userAgentTransport{base: http.DefaultTransport},
		},
	}
}

// SetHTTPClient replaces the service's HTTP client entirely.
// This is primarily used in tests to inject an [httptest.Server] client that
// routes requests to a local test server. Note that the replacement client
// will NOT have the User-Agent transport; use [SetHTTPClientWithUserAgent] if
// the User-Agent header is needed.
func (s *Service) SetHTTPClient(c *http.Client) {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	s.client = c
}

// SetHTTPClientWithUserAgent replaces the service's HTTP client while
// preserving the User-Agent injection behavior. The given client's transport
// is wrapped with [userAgentTransport], so all requests made through the
// returned client will carry the "shu/0.1" User-Agent header.
//
// This is useful in tests where you need both a test-server-routed transport
// and the User-Agent header.
func (s *Service) SetHTTPClientWithUserAgent(c *http.Client) {
	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	s.client = &http.Client{
		Timeout:   c.Timeout,
		Transport: &userAgentTransport{base: transport},
	}
}

func (s *Service) httpClient() *http.Client {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.client
}
