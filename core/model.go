package core

import "time"

type Feed struct {
	ID        int64
	URL       string
	Title     string
	SiteURL   string
	AddedAt   time.Time
	FetchedAt *time.Time
}

type Entry struct {
	ID          int64
	FeedID      int64
	GUID        string
	Title       string
	Link        string
	Summary     string
	PublishedAt *time.Time
	FetchedAt   time.Time
}

type EntryFilter struct {
	FeedID *int64
	Limit  int
	Offset int
}
