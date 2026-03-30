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
	"time"
)

// Store defines the persistence contract required by [Service].
// Implementations must be safe for sequential use from a single goroutine;
// concurrent access is not required by the current design.
type Store interface {
	// AddFeed persists a new feed. On success the feed's ID and AddedAt
	// fields are populated by the store.
	AddFeed(ctx context.Context, feed *Feed) error
	// GetFeed retrieves a single feed by its primary key.
	// It returns an error if the feed does not exist.
	GetFeed(ctx context.Context, id int64) (*Feed, error)
	// ListFeeds returns all registered feeds ordered by ID.
	ListFeeds(ctx context.Context) ([]*Feed, error)
	// RemoveFeed deletes a feed and, via cascade, all of its entries.
	RemoveFeed(ctx context.Context, id int64) error
	// UpdateFeedFetchedAt sets the feed's FetchedAt timestamp to the current
	// time, indicating a successful fetch cycle.
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	// AddEntries inserts entries that do not already exist (deduplicated by
	// the feed_id + GUID pair). It returns the number of newly inserted rows.
	AddEntries(ctx context.Context, entries []*Entry) (int, error)
	// ListEntries returns entries matching the given filter, ordered by
	// fetched_at descending (newest first).
	ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error)
	// Close releases any resources held by the store (e.g. database connections).
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
	store  Store
	logger *slog.Logger
	client *http.Client
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
	s.client = &http.Client{
		Timeout:   c.Timeout,
		Transport: &userAgentTransport{base: transport},
	}
}
