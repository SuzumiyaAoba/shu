// Package core contains the business logic for the shu RSS aggregator.
//
// This package is intentionally free of infrastructure dependencies (no SQLite,
// no CLI framework). It defines its own [Store] interface and operates
// exclusively through that abstraction, allowing the same logic to be driven by
// a CLI, an AWS Lambda handler, or any other composition root.
package core

import (
	"context"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// FeedStore handles feed CRUD operations.
type FeedStore interface {
	AddFeed(ctx context.Context, feed *Feed) error
	GetFeed(ctx context.Context, id int64) (*Feed, error)
	GetFeedByURL(ctx context.Context, url string) (*Feed, error)
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
	ListDeadFeeds(ctx context.Context) ([]*Feed, error)
}

// EntryStore handles entry persistence and queries.
type EntryStore interface {
	AddEntries(ctx context.Context, entries []*Entry) (int, error)
	GetEntry(ctx context.Context, id int64) (*Entry, error)
	ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error)
	CountEntries(ctx context.Context, filter EntryFilter) (int, error)
	SearchEntries(ctx context.Context, query string, limit int) ([]*Entry, error)
	SearchEntriesPage(ctx context.Context, query string, limit, offset int) ([]*Entry, error)
	CountSearchEntries(ctx context.Context, query string) (int, error)
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

// EntryIterator is an optional extension of [EntryStore] that supports
// streaming entry retrieval via a range-over-func iterator. Implementations
// avoid loading the entire result set into memory, which is beneficial for
// large result sets and streaming output.
//
// Use a type assertion to detect support:
//
//	if it, ok := store.(EntryIterator); ok {
//	    for entry, err := range it.IterEntries(ctx, filter) { ... }
//	}
type EntryIterator interface {
	IterEntries(ctx context.Context, filter EntryFilter) iter.Seq2[*Entry, error]
}

// MaintenanceStore provides housekeeping operations.
type MaintenanceStore interface {
	FeedStats(ctx context.Context) ([]FeedStats, error)
	CleanupEntries(ctx context.Context, olderThan time.Time) (int, error)
}

// TxRunner executes fn inside a single database transaction. If the context
// already carries a transaction, the existing one is reused. The transaction
// is rolled back on error and committed on success.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
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
	TxRunner
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

// Service is a backward-compatible facade over the focused domain types in
// this package.
type Service struct {
	feeds       *FeedManager
	fetcher     *Fetcher
	entries     *EntryQueries
	entryState  *EntryStateManager
	tags        *TagManager
	opml        *OPMLHandler
	maintenance *MaintenanceOps
	discovery   *FeedDiscovery
}

// Option customizes a [Service] at construction time.
type Option func(*Service)

// WithHTTPClient configures the service to use the provided HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(s *Service) {
		if c == nil {
			return
		}
		s.setHTTPClient(c)
	}
}

// WithHTTPClientWithUserAgent configures the service to use the provided HTTP
// client while preserving the shu User-Agent header.
func WithHTTPClientWithUserAgent(c *http.Client) Option {
	return func(s *Service) {
		if c == nil {
			return
		}
		s.setHTTPClient(httpClientWithUserAgent(c))
	}
}

// New creates a [Service] with the given store and logger.
// The returned service uses an HTTP client with a 30-second timeout and a
// custom transport that sets the User-Agent header to "shu/0.1".
func New(store Store, logger *slog.Logger, options ...Option) *Service {
	logger = normalizeLogger(logger)
	client := defaultHTTPClient()

	feeds := NewFeedManager(store, logger, client)
	tags := NewTagManager(store, logger)

	svc := &Service{
		feeds:       feeds,
		fetcher:     NewFetcher(store, logger, client),
		entries:     NewEntryQueries(store),
		entryState:  NewEntryStateManager(store),
		tags:        tags,
		opml:        NewOPMLHandler(store, store, feeds, tags, logger),
		maintenance: NewMaintenanceOps(store, store, logger),
		discovery:   NewFeedDiscovery(client),
	}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	return svc
}

func normalizeLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func defaultHTTPClient() *http.Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.Logger = nil // silence default stdlib logger
	rc.HTTPClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: &userAgentTransport{base: http.DefaultTransport},
	}
	return rc.StandardClient()
}

func normalizeHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return defaultHTTPClient()
}

func httpClientWithUserAgent(c *http.Client) *http.Client {
	client := *c
	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = &userAgentTransport{base: transport}
	return &client
}

func (s *Service) setHTTPClient(c *http.Client) {
	s.feeds.setHTTPClient(c)
	s.fetcher.setHTTPClient(c)
	s.discovery.setHTTPClient(c)
}
