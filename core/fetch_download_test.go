package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubDownloadStore struct {
	fetchedIDs   []int64
	recordedID   int64
	recordedMsg  string
	updateErr    error
	recordErr    error
	recordCalled bool
}

func (s *stubDownloadStore) UpdateFeedFetchedAt(_ context.Context, id int64) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.fetchedIDs = append(s.fetchedIDs, id)
	return nil
}

func (s *stubDownloadStore) RecordFeedError(_ context.Context, id int64, errMsg string) error {
	s.recordCalled = true
	s.recordedID = id
	s.recordedMsg = errMsg
	return s.recordErr
}

func TestHTTPFeedDownloaderDownloadReturnsNotModifiedOn304(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != `"etag-1"` {
			t.Fatalf("If-None-Match = %q, want %q", got, `"etag-1"`)
		}
		if got := r.Header.Get("If-Modified-Since"); got != "Fri, 01 Mar 2024 00:00:00 GMT" {
			t.Fatalf("If-Modified-Since = %q", got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer ts.Close()

	store := &stubDownloadStore{}
	downloader := newHTTPFeedDownloader(store, nil, ts.Client())

	document, skipped, err := downloader.download(context.Background(), &Feed{
		ID:           9,
		URL:          ts.URL + "/feed.xml",
		ETag:         `"etag-1"`,
		LastModified: "Fri, 01 Mar 2024 00:00:00 GMT",
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if document != nil {
		t.Fatalf("document = %+v, want nil", document)
	}
	if !skipped {
		t.Fatal("skipped = false, want true")
	}
	if len(store.fetchedIDs) != 1 {
		t.Fatalf("UpdateFeedFetchedAt calls = %d, want 1", len(store.fetchedIDs))
	}
	if store.recordCalled {
		t.Fatal("RecordFeedError should not be called")
	}
}

func TestHTTPFeedDownloaderDownloadRecordsFeedError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	store := &stubDownloadStore{}
	downloader := newHTTPFeedDownloader(store, nil, ts.Client())

	document, skipped, err := downloader.download(context.Background(), &Feed{ID: 3, URL: ts.URL + "/feed.xml"})
	if err == nil {
		t.Fatal("expected error")
	}
	if document != nil {
		t.Fatalf("document = %+v, want nil", document)
	}
	if skipped {
		t.Fatal("skipped = true, want false")
	}
	if !store.recordCalled {
		t.Fatal("RecordFeedError was not called")
	}
	if store.recordedID != 3 {
		t.Fatalf("recordedID = %d, want 3", store.recordedID)
	}
	if store.recordedMsg != "http status 500" {
		t.Fatalf("recordedMsg = %q, want %q", store.recordedMsg, "http status 500")
	}
}

func TestHTTPFeedDownloaderDownloadDoesNotRecordCanceledError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := &stubDownloadStore{}
	downloader := newHTTPFeedDownloader(store, nil, ts.Client())

	_, _, err := downloader.download(ctx, &Feed{ID: 5, URL: ts.URL + "/feed.xml"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if store.recordCalled {
		t.Fatal("RecordFeedError should not be called for canceled context")
	}
}
