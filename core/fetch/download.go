package fetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/SuzumiyaAoba/shu/model"
)

type feedDownloader interface {
	download(ctx context.Context, feed *model.Feed) (*fetchedDocument, bool, error)
}

type feedDownloadStore interface {
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	RecordFeedError(ctx context.Context, id int64, errMsg string) error
}

type httpFeedDownloader struct {
	client *http.Client
	store  feedDownloadStore
	logger *slog.Logger
}

type fetchedDocument struct {
	body     []byte
	headers  http.Header
	finalURL string // URL after following redirects
}

func newHTTPFeedDownloader(store feedDownloadStore, logger *slog.Logger, client *http.Client) *httpFeedDownloader {
	return &httpFeedDownloader{
		client: ensureHTTPClient(client),
		store:  store,
		logger: ensureLogger(logger),
	}
}

func (d *httpFeedDownloader) setHTTPClient(client *http.Client) {
	d.client = ensureHTTPClient(client)
}

// maxFeedBodySize is the upper bound for a downloaded feed document. Feeds
// larger than this limit are truncated, preventing a malicious server from
// exhausting available memory.
const maxFeedBodySize = 10 << 20 // 10 MiB

// FetchBodyConditional downloads a feed document with optional conditional
// GET headers (If-None-Match, If-Modified-Since). Returns nil body on 304.
// The returned finalURL is the URL after following any redirects.
func FetchBodyConditional(ctx context.Context, client *http.Client, url, etag, lastModified string) ([]byte, http.Header, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, "", fmt.Errorf("create request: %w", err)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, "", fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	finalURL := url
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header, finalURL, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBodySize))
	if err != nil {
		return nil, nil, "", fmt.Errorf("read body: %w", err)
	}
	return body, resp.Header, finalURL, nil
}

func (d *httpFeedDownloader) download(ctx context.Context, feed *model.Feed) (*fetchedDocument, bool, error) {
	body, headers, finalURL, err := FetchBodyConditional(ctx, d.client, feed.URL, feed.ETag, feed.LastModified)
	if err != nil {
		return nil, false, d.handleDownloadError(ctx, feed, err)
	}
	if body == nil {
		if err := markFeedFetched(ctx, d.store, feed.ID); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	}
	return &fetchedDocument{body: body, headers: headers, finalURL: finalURL}, false, nil
}

func (d *httpFeedDownloader) handleDownloadError(ctx context.Context, feed *model.Feed, err error) error {
	fetchErr := &model.FeedError{FeedID: feed.ID, FeedURL: feed.URL, Op: "fetch", Err: err}
	if ctx.Err() != nil {
		return fetchErr
	}
	if recErr := d.store.RecordFeedError(ctx, feed.ID, err.Error()); recErr != nil {
		d.logger.With("feed_id", feed.ID).Warn("failed to record feed error", "error", recErr)
	}
	return fetchErr
}
