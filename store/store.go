package store

import (
	"context"

	"github.com/SuzumiyaAoba/shu/core"
)

type Store interface {
	AddFeed(ctx context.Context, feed *core.Feed) error
	GetFeed(ctx context.Context, id int64) (*core.Feed, error)
	ListFeeds(ctx context.Context) ([]*core.Feed, error)
	RemoveFeed(ctx context.Context, id int64) error
	UpdateFeedFetchedAt(ctx context.Context, id int64) error

	AddEntries(ctx context.Context, entries []*core.Entry) (int, error)
	ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error)

	Close() error
}
