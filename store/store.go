// Package store provides the persistence layer for the shu RSS aggregator.
//
// It defines the [Store] interface (mirroring [core.Store]) and provides a
// concrete SQLite implementation via [SQLiteStore]. The package imports
// [core] only for model types ([core.Feed], [core.Entry], [core.EntryFilter])
// and must never depend on the cmd package.
package store

import (
	"context"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

// FeedStore handles feed CRUD operations.
type FeedStore interface {
	AddFeed(ctx context.Context, feed *core.Feed) error
	GetFeed(ctx context.Context, id int64) (*core.Feed, error)
	ListFeeds(ctx context.Context) ([]*core.Feed, error)
	RemoveFeed(ctx context.Context, id int64) error
	UpdateFeed(ctx context.Context, id int64, update core.FeedUpdate) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error
}

// FeedHealthStore tracks feed fetch errors and disabled state.
type FeedHealthStore interface {
	RecordFeedError(ctx context.Context, id int64, errMsg string) error
	ResetFeedError(ctx context.Context, id int64) error
	SetFeedDisabled(ctx context.Context, id int64, disabled bool) error
}

// EntryStore handles entry persistence and queries.
type EntryStore interface {
	AddEntries(ctx context.Context, entries []*core.Entry) (int, error)
	GetEntry(ctx context.Context, id int64) (*core.Entry, error)
	ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error)
	SearchEntries(ctx context.Context, query string, limit int) ([]*core.Entry, error)
	FindDuplicateEntries(ctx context.Context, entryID int64) ([]*core.Entry, error)
}

// EntryStateStore manages read/star state on entries.
type EntryStateStore interface {
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	StarEntry(ctx context.Context, id int64) error
	UnstarEntry(ctx context.Context, id int64) error
}

// TagStore handles tag CRUD and feed-tag associations.
type TagStore interface {
	AddTag(ctx context.Context, feedID int64, tagName string) error
	RemoveTag(ctx context.Context, feedID int64, tagName string) error
	ListTags(ctx context.Context, feedID int64) ([]core.Tag, error)
	ListAllTags(ctx context.Context) ([]core.Tag, error)
	ListFeedsByTag(ctx context.Context, tagName string) ([]*core.Feed, error)
}

// MaintenanceStore provides housekeeping operations.
type MaintenanceStore interface {
	FeedStats(ctx context.Context) ([]core.FeedStats, error)
	CleanupEntries(ctx context.Context, olderThan time.Time) (int, error)
}

// Store is the full persistence interface satisfied by [SQLiteStore].
// It mirrors [core.Store] and composes all role-specific interfaces.
type Store interface {
	FeedStore
	FeedHealthStore
	EntryStore
	EntryStateStore
	TagStore
	MaintenanceStore
	Close() error
}
