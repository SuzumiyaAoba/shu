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

	// Close releases database connections and other resources.
	Close() error
}
