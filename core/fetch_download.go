package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type fetchedFeedDocument struct {
	body    []byte
	headers http.Header
}

// fetchBody downloads the feed document at the given URL and returns the raw
// response body. This is used by AddFeed where no conditional GET is needed.
func (s *Service) fetchBody(ctx context.Context, url string) ([]byte, error) {
	body, _, err := s.fetchBodyConditional(ctx, url, "", "")
	return body, err
}

// fetchBodyConditional downloads the feed document with optional conditional
// GET headers (If-None-Match, If-Modified-Since). Returns nil body on 304.
func (s *Service) fetchBodyConditional(ctx context.Context, url, etag, lastModified string) ([]byte, http.Header, error) {
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

	resp, err := s.httpClient().Do(req)
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

func (s *Service) downloadFeedDocument(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error) {
	body, headers, err := s.fetchBodyConditional(ctx, feed.URL, feed.ETag, feed.LastModified)
	if err != nil {
		return nil, false, s.handleFeedDownloadError(ctx, feed, err)
	}
	if body == nil {
		if err := s.markFeedFetched(ctx, feed.ID); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	}
	return &fetchedFeedDocument{body: body, headers: headers}, false, nil
}

func (s *Service) handleFeedDownloadError(ctx context.Context, feed *Feed, err error) error {
	fetchErr := fmt.Errorf("fetch feed %s: %w", feed.URL, err)
	if ctx.Err() != nil {
		return fetchErr
	}
	if recErr := s.store.RecordFeedError(ctx, feed.ID, err.Error()); recErr != nil {
		s.logger.Warn("failed to record feed error", "id", feed.ID, "error", recErr)
	}
	return fetchErr
}
