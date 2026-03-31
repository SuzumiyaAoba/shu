// Package store provides the persistence layer for the shu RSS aggregator.
//
// It defines the [Store] interface (mirroring [core.Store]) and provides a
// concrete SQLite implementation via [SQLiteStore]. The package imports
// [core] only for model types ([core.Feed], [core.Entry], [core.EntryFilter])
// and must never depend on the cmd package.
package store

import (
	"context"

	"github.com/SuzumiyaAoba/shu/core"
)

// Store defines the data access interface for feeds and entries.
// This interface mirrors [core.Store] and is satisfied by [SQLiteStore].
// It exists in the store package to allow type-asserting or referencing the
// store-specific interface when needed, while the core package depends only on
// its own copy of the interface.
type Store interface {
	// AddFeed persists a new feed record. On success, the feed's ID and
	// AddedAt fields are populated in place.
	AddFeed(ctx context.Context, feed *core.Feed) error
	// GetFeed retrieves a feed by its primary key. Returns an error if not found.
	GetFeed(ctx context.Context, id int64) (*core.Feed, error)
	// ListFeeds returns all feeds ordered by ID.
	ListFeeds(ctx context.Context) ([]*core.Feed, error)
	// RemoveFeed deletes a feed by ID. Associated entries are cascade-deleted
	// by the database's ON DELETE CASCADE constraint.
	RemoveFeed(ctx context.Context, id int64) error
	// UpdateFeedFetchedAt sets the feed's fetched_at column to the current
	// UTC time, recording a successful fetch cycle.
	UpdateFeedFetchedAt(ctx context.Context, id int64) error

	// AddEntries bulk-inserts entries, silently skipping any that already
	// exist (deduplicated by the UNIQUE(feed_id, guid) constraint). Returns
	// the count of newly inserted rows.
	AddEntries(ctx context.Context, entries []*core.Entry) (int, error)
	// ListEntries queries entries matching the filter. Results are ordered by
	// fetched_at descending (newest first).
	ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error)

	// UpdateFeed updates mutable feed fields.
	UpdateFeed(ctx context.Context, id int64, update core.FeedUpdate) error
	// UpdateFeedCacheHeaders stores HTTP cache headers for conditional GET.
	UpdateFeedCacheHeaders(ctx context.Context, id int64, etag, lastModified string) error

	// MarkEntryRead sets the read_at timestamp on the given entry.
	MarkEntryRead(ctx context.Context, id int64) error
	// MarkEntryUnread clears the read_at timestamp on the given entry.
	MarkEntryUnread(ctx context.Context, id int64) error

	// AddTag creates a tag and associates it with a feed.
	AddTag(ctx context.Context, feedID int64, tagName string) error
	// RemoveTag removes a tag association from a feed.
	RemoveTag(ctx context.Context, feedID int64, tagName string) error
	// ListTags returns all tags for a feed.
	ListTags(ctx context.Context, feedID int64) ([]core.Tag, error)
	// ListAllTags returns every tag.
	ListAllTags(ctx context.Context) ([]core.Tag, error)
	// ListFeedsByTag returns feeds with the given tag.
	ListFeedsByTag(ctx context.Context, tagName string) ([]*core.Feed, error)

	// SearchEntries performs full-text search on entries.
	SearchEntries(ctx context.Context, query string, limit int) ([]*core.Entry, error)

	// Close releases database connections and other resources.
	Close() error
}
