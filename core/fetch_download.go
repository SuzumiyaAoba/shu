package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type feedDownloader interface {
	download(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error)
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

type fetchedFeedDocument struct {
	body    []byte
	headers http.Header
}

func newHTTPFeedDownloader(store feedDownloadStore, logger *slog.Logger, client *http.Client) *httpFeedDownloader {
	return &httpFeedDownloader{
		client: normalizeHTTPClient(client),
		store:  store,
		logger: normalizeLogger(logger),
	}
}

func (d *httpFeedDownloader) setHTTPClient(client *http.Client) {
	d.client = normalizeHTTPClient(client)
}

// fetchBody downloads the feed document at the given URL and returns the raw
// response body. This is used by AddFeed where no conditional GET is needed.
func fetchBody(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	body, _, err := fetchBodyConditional(ctx, client, url, "", "")
	return body, err
}

// fetchBodyConditional downloads the feed document with optional conditional
// GET headers (If-None-Match, If-Modified-Since). Returns nil body on 304.
func fetchBodyConditional(ctx context.Context, client *http.Client, url, etag, lastModified string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}
	return body, resp.Header, nil
}

func (d *httpFeedDownloader) download(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error) {
	body, headers, err := fetchBodyConditional(ctx, d.client, feed.URL, feed.ETag, feed.LastModified)
	if err != nil {
		return nil, false, d.handleFeedDownloadError(ctx, feed, err)
	}
	if body == nil {
		if err := markFeedFetched(ctx, d.store, feed.ID); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	}
	return &fetchedFeedDocument{body: body, headers: headers}, false, nil
}

func (d *httpFeedDownloader) handleFeedDownloadError(ctx context.Context, feed *Feed, err error) error {
	fetchErr := &FeedError{FeedID: feed.ID, FeedURL: feed.URL, Op: "fetch", Err: err}
	if ctx.Err() != nil {
		return fetchErr
	}
	if recErr := d.store.RecordFeedError(ctx, feed.ID, err.Error()); recErr != nil {
		d.logger.With("feed_id", feed.ID).Warn("failed to record feed error", "error", recErr)
	}
	return fetchErr
}
