package cmd

import (
	"context"
	"io"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

type feedService interface {
	AddFeed(ctx context.Context, url string, titleOverride string) (*core.Feed, error)
	ListFeeds(ctx context.Context) ([]*core.Feed, error)
	FetchFeed(ctx context.Context, feedID int64) ([]*core.Entry, error)
	FetchAll(ctx context.Context) (int, error)
	DiscoverFeeds(ctx context.Context, pageURL string) ([]string, error)
	UpdateFeed(ctx context.Context, id int64, update core.FeedUpdate) error
	RemoveFeed(ctx context.Context, id int64) error
	EnableFeed(ctx context.Context, id int64) error
	DisableFeed(ctx context.Context, id int64) error
}

type entryService interface {
	GetEntry(ctx context.Context, id int64) (*core.Entry, error)
	ListEntries(ctx context.Context, filter core.EntryFilter) ([]*core.Entry, error)
	ListEntriesPage(ctx context.Context, filter core.EntryFilter) (*core.EntryPage, error)
	SearchEntries(ctx context.Context, query string, limit int) ([]*core.Entry, error)
	SearchEntriesPage(ctx context.Context, query string, limit, offset int) (*core.EntryPage, error)
	FindDuplicateEntries(ctx context.Context, entryID int64) ([]*core.Entry, error)
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntriesRead(ctx context.Context, ids []int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	MarkEntriesUnread(ctx context.Context, ids []int64) error
	StarEntry(ctx context.Context, id int64) error
	StarEntries(ctx context.Context, ids []int64) error
	UnstarEntry(ctx context.Context, id int64) error
	UnstarEntries(ctx context.Context, ids []int64) error
}

type tagService interface {
	AddTag(ctx context.Context, feedID int64, tagName string) error
	RemoveTag(ctx context.Context, feedID int64, tagName string) error
	ListTags(ctx context.Context, feedID int64) ([]core.Tag, error)
	ListAllTags(ctx context.Context) ([]core.Tag, error)
}

type maintenanceService interface {
	FeedStatsAll(ctx context.Context) ([]core.FeedStats, error)
	CleanupEntries(ctx context.Context, olderThan time.Duration) (int, error)
	FetchAll(ctx context.Context) (int, error)
}

type opmlService interface {
	ImportOPML(ctx context.Context, r io.Reader) (int, error)
	ImportOPMLDetailed(ctx context.Context, r io.Reader) (*core.OPMLImportResult, error)
	ExportOPML(ctx context.Context) (*core.OPML, error)
}

type feedServiceGetter func() (feedService, error)
type entryServiceGetter func() (entryService, error)
type tagServiceGetter func() (tagService, error)
type maintenanceServiceGetter func() (maintenanceService, error)
type opmlServiceGetter func() (opmlService, error)
