package store

import (
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/SuzumiyaAoba/shu/core"
)

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

	e.Categories = toNilIfEmpty(categories)
	e.Enclosures = toNilIfEmpty(enclosures)
	e.Authors = toNilIfEmpty(authors)
	e.Links = toNilIfEmpty(links)
	e.Contributors = toNilIfEmpty(contributors)
	e.Source = toNilIfEmpty(source)

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

// iterRows returns an [iter.Seq2] iterator over database rows. Each call to
// yield delivers the next scanned value and any scan error. The caller must
// return false from yield to stop early; the rows are closed when iteration
// ends or the iterator function returns.
//
// This is the streaming counterpart to [collectRows]: use it when you want to
// process rows one-at-a-time without loading the full result set into memory.
func iterRows[T any](rows *sql.Rows, scan func(scanner) (*T, error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			v, err := scan(rows)
			if !yield(v, err) {
				return
			}
			if err != nil {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// collectRows iterates all rows, scans each into a *T via scan, and returns the
// collected slice. It closes rows on return. This is a generic helper that
// eliminates the identical boilerplate present in collectFeeds, collectEntries,
// etc.
func collectRows[T any](rows *sql.Rows, scan func(scanner) (*T, error)) ([]*T, error) {
	defer func() { _ = rows.Close() }()

	var result []*T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// collectValues is like collectRows but for value types (non-pointer).
func collectValues[T any](rows *sql.Rows, scan func(scanner) (T, error)) ([]T, error) {
	defer func() { _ = rows.Close() }()

	var result []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func collectFeeds(rows *sql.Rows) ([]*core.Feed, error) {
	return collectRows(rows, scanFeed)
}

func collectEntries(rows *sql.Rows) ([]*core.Entry, error) {
	return collectRows(rows, scanEntry)
}

func toNilIfEmpty(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

func scanTag(s scanner) (core.Tag, error) {
	var t core.Tag
	if err := s.Scan(&t.ID, &t.Name); err != nil {
		return core.Tag{}, fmt.Errorf("scan tag: %w", err)
	}
	return t, nil
}

func collectTags(rows *sql.Rows) ([]core.Tag, error) {
	return collectValues(rows, scanTag)
}
