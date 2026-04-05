package core

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type stubPersistStore struct {
	addEntriesCalls   int
	addedEntries      []*Entry
	inserted          int
	addErr            error
	fetchedIDs        []int64
	updateFetchedErr  error
	resetIDs          []int64
	resetErr          error
	cacheHeaderFeedID int64
	cacheHeaderETag   string
	cacheHeaderLast   string
	cacheHeaderErr    error
}

func (s *stubPersistStore) AddEntries(_ context.Context, entries []*Entry) (int, error) {
	s.addEntriesCalls++
	s.addedEntries = entries
	if s.addErr != nil {
		return 0, s.addErr
	}
	return s.inserted, nil
}

func (s *stubPersistStore) UpdateFeedFetchedAt(_ context.Context, id int64) error {
	if s.updateFetchedErr != nil {
		return s.updateFetchedErr
	}
	s.fetchedIDs = append(s.fetchedIDs, id)
	return nil
}

func (s *stubPersistStore) ResetFeedError(_ context.Context, id int64) error {
	s.resetIDs = append(s.resetIDs, id)
	return s.resetErr
}

func (s *stubPersistStore) UpdateFeedCacheHeaders(_ context.Context, id int64, etag, lastModified string) error {
	s.cacheHeaderFeedID = id
	s.cacheHeaderETag = etag
	s.cacheHeaderLast = lastModified
	return s.cacheHeaderErr
}

func (s *stubPersistStore) UpdateFeed(_ context.Context, _ int64, _ FeedUpdate) error {
	return nil
}

func (s *stubPersistStore) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestStoreFeedPersisterPersistStoresEntriesAndUpdatesFeedState(t *testing.T) {
	store := &stubPersistStore{inserted: 2}
	persister := newStoreFeedPersister(store, nil)
	headers := http.Header{}
	headers.Set("ETag", `"etag-1"`)
	headers.Set("Last-Modified", "Fri, 01 Mar 2024 00:00:00 GMT")

	result, err := persister.persist(context.Background(), &Feed{
		ID:    11,
		URL:   "https://example.com/feed.xml",
		Title: "Test Feed",
	}, &fetchedFeedDocument{
		body:    []byte(parseTestRSSFeed),
		headers: headers,
	})
	if err != nil {
		t.Fatalf("persist failed: %v", err)
	}
	if result.inserted != 2 {
		t.Fatalf("inserted = %d, want 2", result.inserted)
	}
	if len(result.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(result.entries))
	}
	if store.addEntriesCalls != 1 {
		t.Fatalf("AddEntries calls = %d, want 1", store.addEntriesCalls)
	}
	if len(store.addedEntries) != 2 {
		t.Fatalf("AddEntries received %d entries, want 2", len(store.addedEntries))
	}
	if len(store.fetchedIDs) != 1 || store.fetchedIDs[0] != 11 {
		t.Fatalf("UpdateFeedFetchedAt calls = %+v, want [11]", store.fetchedIDs)
	}
	if len(store.resetIDs) != 1 || store.resetIDs[0] != 11 {
		t.Fatalf("ResetFeedError calls = %+v, want [11]", store.resetIDs)
	}
	if store.cacheHeaderFeedID != 11 {
		t.Fatalf("cache header feed ID = %d, want 11", store.cacheHeaderFeedID)
	}
	if store.cacheHeaderETag != `"etag-1"` {
		t.Fatalf("cache header etag = %q", store.cacheHeaderETag)
	}
	if store.cacheHeaderLast != "Fri, 01 Mar 2024 00:00:00 GMT" {
		t.Fatalf("cache header last-modified = %q", store.cacheHeaderLast)
	}
}

func TestStoreFeedPersisterPersistReturnsParseError(t *testing.T) {
	store := &stubPersistStore{}
	persister := newStoreFeedPersister(store, nil)

	_, err := persister.persist(context.Background(), &Feed{
		ID:  1,
		URL: "https://example.com/feed.xml",
	}, &fetchedFeedDocument{
		body:    []byte("not a feed"),
		headers: http.Header{},
	})
	if !errors.Is(err, ErrInvalidFeed) {
		t.Fatalf("expected ErrInvalidFeed, got %v", err)
	}
	if store.addEntriesCalls != 0 {
		t.Fatalf("AddEntries calls = %d, want 0", store.addEntriesCalls)
	}
}

func TestStoreFeedPersisterPersistIgnoresResetFeedErrorFailure(t *testing.T) {
	store := &stubPersistStore{
		inserted: 1,
		resetErr: errors.New("reset failed"),
	}
	persister := newStoreFeedPersister(store, nil)

	result, err := persister.persist(context.Background(), &Feed{
		ID:  4,
		URL: "https://example.com/feed.xml",
	}, &fetchedFeedDocument{
		body:    []byte(parseTestRSSFeed),
		headers: http.Header{},
	})
	if err != nil {
		t.Fatalf("persist failed: %v", err)
	}
	if result.inserted != 1 {
		t.Fatalf("inserted = %d, want 1", result.inserted)
	}
	if len(store.resetIDs) != 1 || store.resetIDs[0] != 4 {
		t.Fatalf("ResetFeedError calls = %+v, want [4]", store.resetIDs)
	}
}
