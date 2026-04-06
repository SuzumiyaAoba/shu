package cmd

import (
	"context"
	"io"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/core/fetch"
	"github.com/SuzumiyaAoba/shu/model"
)

type feedService interface {
	AddFeed(ctx context.Context, url string, titleOverride string) (*model.Feed, error)
	ListDeadFeeds(ctx context.Context) ([]*model.Feed, error)
	ListFeeds(ctx context.Context) ([]*model.Feed, error)
	FetchFeed(ctx context.Context, feedID int64) ([]*model.Entry, error)
	FetchAll(ctx context.Context) (int, error)
	FetchAllWithObserver(ctx context.Context, observer fetch.Observer) (int, error)
	DiscoverFeeds(ctx context.Context, pageURL string) ([]string, error)
	UpdateFeed(ctx context.Context, id int64, update model.FeedUpdate) error
	RemoveFeed(ctx context.Context, id int64) error
	EnableFeed(ctx context.Context, id int64) error
	DisableFeed(ctx context.Context, id int64) error
	RemoveDeadFeeds(ctx context.Context) ([]*model.Feed, error)
}

type entryService interface {
	GetEntry(ctx context.Context, id int64) (*model.Entry, error)
	ListEntries(ctx context.Context, filter model.EntryFilter) ([]*model.Entry, error)
	ListEntriesPage(ctx context.Context, filter model.EntryFilter) (*model.EntryPage, error)
	SearchEntries(ctx context.Context, query string, limit int) ([]*model.Entry, error)
	SearchEntriesPage(ctx context.Context, query string, limit, offset int) (*model.EntryPage, error)
	FindDuplicateEntries(ctx context.Context, entryID int64) ([]*model.Entry, error)
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
	ListTags(ctx context.Context, feedID int64) ([]model.Tag, error)
	ListAllTags(ctx context.Context) ([]model.Tag, error)
}

type maintenanceService interface {
	FeedStatsAll(ctx context.Context) ([]model.FeedStats, error)
	CleanupEntries(ctx context.Context, olderThan time.Duration) (int, error)
	FetchAll(ctx context.Context) (int, error)
}

type opmlService interface {
	ImportOPML(ctx context.Context, r io.Reader) (int, error)
	ImportOPMLDetailed(ctx context.Context, r io.Reader) (*core.OPMLImportResult, error)
	ExportOPML(ctx context.Context) (*core.OPML, error)
	FetchAll(ctx context.Context) (int, error)
}

type feedServiceGetter func() (feedService, error)
type entryServiceGetter func() (entryService, error)
type tagServiceGetter func() (tagService, error)
type maintenanceServiceGetter func() (maintenanceService, error)
type opmlServiceGetter func() (opmlService, error)
