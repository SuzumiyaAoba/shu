package core

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type Store interface {
	AddFeed(ctx context.Context, feed *Feed) error
	GetFeed(ctx context.Context, id int64) (*Feed, error)
	ListFeeds(ctx context.Context) ([]*Feed, error)
	RemoveFeed(ctx context.Context, id int64) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	AddEntries(ctx context.Context, entries []*Entry) (int, error)
	ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error)
	Close() error
}

const userAgent = "shu/0.1"

type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent)
	return t.base.RoundTrip(req)
}

type Service struct {
	store  Store
	logger *slog.Logger
	client *http.Client
}

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

func (s *Service) SetHTTPClient(c *http.Client) {
	s.client = c
}

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
