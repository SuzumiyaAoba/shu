package coretest

import (
	"context"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/model"
)

// BaseFakeStore provides no-op implementations for the full core.Store contract.
var _ core.Store = BaseFakeStore{}

type BaseFakeStore struct{}

func (BaseFakeStore) AddFeed(context.Context, *model.Feed) error { return nil }

func (BaseFakeStore) GetFeed(context.Context, int64) (*model.Feed, error) { return nil, nil }

func (BaseFakeStore) GetFeedByURL(context.Context, string) (*model.Feed, error) { return nil, nil }

func (BaseFakeStore) ListFeeds(context.Context) ([]*model.Feed, error) { return nil, nil }

func (BaseFakeStore) RemoveFeed(context.Context, int64) error { return nil }

func (BaseFakeStore) UpdateFeed(context.Context, int64, model.FeedUpdate) error { return nil }

func (BaseFakeStore) UpdateFeedFetchedAt(context.Context, int64) error { return nil }

func (BaseFakeStore) UpdateFeedCacheHeaders(context.Context, int64, string, string) error {
	return nil
}

func (BaseFakeStore) RecordFeedError(context.Context, int64, string) error { return nil }

func (BaseFakeStore) ResetFeedError(context.Context, int64) error { return nil }

func (BaseFakeStore) SetFeedDisabled(context.Context, int64, bool) error { return nil }

func (BaseFakeStore) ListDeadFeeds(context.Context) ([]*model.Feed, error) { return nil, nil }

func (BaseFakeStore) AddEntries(context.Context, []*model.Entry) (int, error) { return 0, nil }

func (BaseFakeStore) GetEntry(context.Context, int64) (*model.Entry, error) { return nil, nil }

func (BaseFakeStore) ListEntries(context.Context, model.EntryFilter) ([]*model.Entry, error) {
	return nil, nil
}

func (BaseFakeStore) CountEntries(context.Context, model.EntryFilter) (int, error) { return 0, nil }

func (BaseFakeStore) SearchEntries(context.Context, string, int) ([]*model.Entry, error) {
	return nil, nil
}

func (BaseFakeStore) SearchEntriesPage(context.Context, string, int, int) ([]*model.Entry, error) {
	return nil, nil
}

func (BaseFakeStore) CountSearchEntries(context.Context, string) (int, error) { return 0, nil }

func (BaseFakeStore) FindDuplicateEntries(context.Context, int64) ([]*model.Entry, error) {
	return nil, nil
}

func (BaseFakeStore) MarkEntryRead(context.Context, int64) error { return nil }

func (BaseFakeStore) MarkEntriesRead(context.Context, []int64) error { return nil }

func (BaseFakeStore) MarkEntryUnread(context.Context, int64) error { return nil }

func (BaseFakeStore) MarkEntriesUnread(context.Context, []int64) error { return nil }

func (BaseFakeStore) StarEntry(context.Context, int64) error { return nil }

func (BaseFakeStore) StarEntries(context.Context, []int64) error { return nil }

func (BaseFakeStore) UnstarEntry(context.Context, int64) error { return nil }

func (BaseFakeStore) UnstarEntries(context.Context, []int64) error { return nil }

func (BaseFakeStore) AddTag(context.Context, int64, string) error { return nil }

func (BaseFakeStore) RemoveTag(context.Context, int64, string) error { return nil }

func (BaseFakeStore) ListTags(context.Context, int64) ([]model.Tag, error) { return nil, nil }

func (BaseFakeStore) ListFeedTags(context.Context) (map[int64][]model.Tag, error) { return nil, nil }

func (BaseFakeStore) ListAllTags(context.Context) ([]model.Tag, error) { return nil, nil }

func (BaseFakeStore) ListFeedsByTag(context.Context, string) ([]*model.Feed, error) { return nil, nil }

func (BaseFakeStore) FeedStats(context.Context) ([]model.FeedStats, error) { return nil, nil }

func (BaseFakeStore) CleanupEntries(context.Context, time.Time) (int, error) { return 0, nil }

func (BaseFakeStore) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (BaseFakeStore) Close() error { return nil }
