package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

const sqliteConstraintUnique = 2067

// feedColumns is the SELECT column list shared by GetFeed and ListFeeds.
const feedColumns = `id, url, title, site_url, added_at, fetched_at, description, language, image_url, feed_type, etag, last_modified, error_count, last_error, disabled, fetch_interval_sec`

// entryColumns is the SELECT column list shared by ListEntries.
const entryColumns = `id, feed_id, guid, title, link, summary, published_at, fetched_at, content, author, image_url, categories, updated_at, enclosures, authors, links, contributors, rights, source, read_at, starred_at`

const insertEntrySQL = `INSERT OR IGNORE INTO entries (feed_id, guid, title, link, summary, published_at, content, author, image_url, categories, updated_at, enclosures, authors, links, contributors, rights, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// nowRFC3339 returns the current UTC time formatted as RFC 3339.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// parseNullableTime parses a nullable RFC 3339 timestamp string into *time.Time.
func parseNullableTime(s *string, field string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	return &t, nil
}

// formatNullableTime formats a *time.Time as an RFC 3339 string pointer.
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// scanner is a minimal interface satisfied by both *sql.Row and *sql.Rows,
// allowing scanFeed and scanEntry to work with either single-row or multi-row
// query results.
type scanner interface {
	Scan(dest ...any) error
}

// scanFeed reads a single feed row from the scanner and converts the stored
// ISO 8601 timestamp strings back into Go time.Time values.
func scanFeed(s scanner) (*core.Feed, error) {
	var f core.Feed
	var addedAt string
	var fetchedAt *string

	var disabled int
	if err := s.Scan(
		&f.ID, &f.URL, &f.Title, &f.SiteURL, &addedAt, &fetchedAt,
		&f.Description, &f.Language, &f.ImageURL, &f.FeedType,
		&f.ETag, &f.LastModified,
		&f.ErrorCount, &f.LastError, &disabled, &f.FetchIntervalSec,
	); err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}

	t, err := time.Parse(time.RFC3339, addedAt)
	if err != nil {
		return nil, fmt.Errorf("parse added_at: %w", err)
	}
	f.AddedAt = t

	if f.FetchedAt, err = parseNullableTime(fetchedAt, "fetched_at"); err != nil {
		return nil, err
	}

	f.Disabled = disabled != 0

	return &f, nil
}

// scanEntry reads a single entry row from the scanner and converts the stored
// ISO 8601 timestamp strings back into Go time.Time values. The published_at
// and updated_at columns are nullable and map to *time.Time.
func scanEntry(s scanner) (*core.Entry, error) {
	var e core.Entry
	var publishedAt *string
	var fetchedAt string
	var updatedAt *string
	var readAt *string
	var starredAt *string

	var categories, enclosures, authors, links, contributors, source string

	if err := s.Scan(
		&e.ID, &e.FeedID, &e.GUID, &e.Title, &e.Link, &e.Summary,
		&publishedAt, &fetchedAt,
		&e.Content, &e.Author, &e.ImageURL, &categories, &updatedAt, &enclosures,
		&authors, &links, &contributors, &e.Rights, &source,
		&readAt, &starredAt,
	); err != nil {
		return nil, fmt.Errorf("scan entry: %w", err)
	}

	t, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return nil, fmt.Errorf("parse fetched_at: %w", err)
	}
	e.FetchedAt = t

	if e.PublishedAt, err = parseNullableTime(publishedAt, "published_at"); err != nil {
		return nil, err
	}
	if e.UpdatedAt, err = parseNullableTime(updatedAt, "updated_at"); err != nil {
		return nil, err
	}
	if e.ReadAt, err = parseNullableTime(readAt, "read_at"); err != nil {
		return nil, err
	}
	if e.StarredAt, err = parseNullableTime(starredAt, "starred_at"); err != nil {
		return nil, err
	}

	e.Categories = []byte(categories)
	if len(e.Categories) == 0 {
		e.Categories = nil
	}
	e.Enclosures = []byte(enclosures)
	if len(e.Enclosures) == 0 {
		e.Enclosures = nil
	}
	e.Authors = []byte(authors)
	if len(e.Authors) == 0 {
		e.Authors = nil
	}
	e.Links = []byte(links)
	if len(e.Links) == 0 {
		e.Links = nil
	}
	e.Contributors = []byte(contributors)
	if len(e.Contributors) == 0 {
		e.Contributors = nil
	}
	e.Source = []byte(source)
	if len(e.Source) == 0 {
		e.Source = nil
	}

	return &e, nil
}

func fetchFeed(s scanner, label string) (*core.Feed, error) {
	feed, err := scanFeed(s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", label, core.ErrFeedNotFound)
		}
		return nil, err
	}
	return feed, nil
}

func fetchEntry(s scanner, label string) (*core.Entry, error) {
	entry, err := scanEntry(s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", label, core.ErrEntryNotFound)
		}
		return nil, err
	}
	return entry, nil
}

func collectFeeds(rows *sql.Rows) ([]*core.Feed, error) {
	defer func() { _ = rows.Close() }()

	var feeds []*core.Feed
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

func collectEntries(rows *sql.Rows) ([]*core.Entry, error) {
	defer func() { _ = rows.Close() }()

	var entries []*core.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func collectTags(rows *sql.Rows) ([]core.Tag, error) {
	defer func() { _ = rows.Close() }()

	var tags []core.Tag
	for rows.Next() {
		var t core.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
