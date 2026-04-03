package app

import (
	"context"
	"net/http"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type fakeStore struct{}

func newFakeStore() *fakeStore { return &fakeStore{} }

func (f *fakeStore) AddFeed(context.Context, *core.Feed) error                           { return nil }
func (f *fakeStore) GetFeed(context.Context, int64) (*core.Feed, error)                  { return nil, nil }
func (f *fakeStore) GetFeedByURL(context.Context, string) (*core.Feed, error)            { return nil, nil }
func (f *fakeStore) ListFeeds(context.Context) ([]*core.Feed, error)                     { return nil, nil }
func (f *fakeStore) RemoveFeed(context.Context, int64) error                             { return nil }
func (f *fakeStore) UpdateFeed(context.Context, int64, core.FeedUpdate) error            { return nil }
func (f *fakeStore) UpdateFeedFetchedAt(context.Context, int64) error                    { return nil }
func (f *fakeStore) UpdateFeedCacheHeaders(context.Context, int64, string, string) error { return nil }
func (f *fakeStore) RecordFeedError(context.Context, int64, string) error                { return nil }
func (f *fakeStore) ResetFeedError(context.Context, int64) error                         { return nil }
func (f *fakeStore) SetFeedDisabled(context.Context, int64, bool) error                  { return nil }
func (f *fakeStore) AddEntries(context.Context, []*core.Entry) (int, error)              { return 0, nil }
func (f *fakeStore) GetEntry(context.Context, int64) (*core.Entry, error)                { return nil, nil }
func (f *fakeStore) ListEntries(context.Context, core.EntryFilter) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) CountEntries(context.Context, core.EntryFilter) (int, error) { return 0, nil }
func (f *fakeStore) SearchEntries(context.Context, string, int) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) SearchEntriesPage(context.Context, string, int, int) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) CountSearchEntries(context.Context, string) (int, error) { return 0, nil }
func (f *fakeStore) FindDuplicateEntries(context.Context, int64) ([]*core.Entry, error) {
	return nil, nil
}
func (f *fakeStore) MarkEntryRead(context.Context, int64) error                   { return nil }
func (f *fakeStore) MarkEntriesRead(context.Context, []int64) error               { return nil }
func (f *fakeStore) MarkEntryUnread(context.Context, int64) error                 { return nil }
func (f *fakeStore) MarkEntriesUnread(context.Context, []int64) error             { return nil }
func (f *fakeStore) StarEntry(context.Context, int64) error                       { return nil }
func (f *fakeStore) StarEntries(context.Context, []int64) error                   { return nil }
func (f *fakeStore) UnstarEntry(context.Context, int64) error                     { return nil }
func (f *fakeStore) UnstarEntries(context.Context, []int64) error                 { return nil }
func (f *fakeStore) AddTag(context.Context, int64, string) error                  { return nil }
func (f *fakeStore) RemoveTag(context.Context, int64, string) error               { return nil }
func (f *fakeStore) ListTags(context.Context, int64) ([]core.Tag, error)          { return nil, nil }
func (f *fakeStore) ListFeedTags(context.Context) (map[int64][]core.Tag, error)   { return nil, nil }
func (f *fakeStore) ListAllTags(context.Context) ([]core.Tag, error)              { return nil, nil }
func (f *fakeStore) ListFeedsByTag(context.Context, string) ([]*core.Feed, error) { return nil, nil }
func (f *fakeStore) FeedStats(context.Context) ([]core.FeedStats, error)          { return nil, nil }
func (f *fakeStore) CleanupEntries(context.Context, time.Time) (int, error)       { return 0, nil }
func (f *fakeStore) Close() error                                                 { return nil }

type trackingStore struct {
	*fakeStore
	closeFn func() error
}

func (s *trackingStore) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}
