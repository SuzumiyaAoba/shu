package core

import (
	"errors"
	"fmt"
	"log/slog"
)

var (
	// ErrFeedAlreadyExists indicates a feed with the same URL is already registered.
	ErrFeedAlreadyExists = errors.New("feed already exists")
	// ErrInvalidFeed indicates the fetched document is not a valid feed.
	ErrInvalidFeed = errors.New("invalid feed")
	// ErrFeedNotFound indicates the requested feed does not exist.
	ErrFeedNotFound = errors.New("feed not found")
	// ErrEntryNotFound indicates the requested entry does not exist.
	ErrEntryNotFound = errors.New("entry not found")
	// ErrInvalidOPML indicates the provided OPML document is invalid.
	ErrInvalidOPML = errors.New("invalid opml")
	// ErrTagApplyFailed indicates a tag operation failed while applying metadata.
	ErrTagApplyFailed = errors.New("tag apply failed")
)

// FeedError is a structured error that carries context about which feed
// operation failed. It wraps the underlying cause so callers can use
// errors.Is / errors.As to inspect the full chain.
type FeedError struct {
	// FeedID is the database primary key of the affected feed.
	FeedID int64
	// FeedURL is the URL of the affected feed.
	FeedURL string
	// Op is the operation that failed (e.g. "fetch", "add").
	Op string
	// Err is the underlying error.
	Err error
}

func (e *FeedError) Error() string {
	return fmt.Sprintf("%s feed %d (%s): %v", e.Op, e.FeedID, e.FeedURL, e.Err)
}

// Unwrap exposes the underlying error so errors.Is / errors.As traverse it.
func (e *FeedError) Unwrap() error { return e.Err }

// LogValue implements [slog.LogValuer] so a *FeedError emits structured fields
// when logged via slog rather than a single string.
func (e *FeedError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("feed_id", e.FeedID),
		slog.String("feed_url", e.FeedURL),
		slog.String("op", e.Op),
		slog.Any("cause", e.Err),
	)
}

// StoreError is a structured error that carries context about which storage
// operation failed and on which table. It wraps the underlying cause so callers
// can use errors.Is / errors.As to inspect the full chain.
type StoreError struct {
	// Op is the storage operation that failed (e.g. "add", "list", "get").
	Op string
	// Table is the database table involved (e.g. "feeds", "entries", "tags").
	Table string
	// Err is the underlying error.
	Err error
}

func (e *StoreError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Table, e.Err)
}

// Unwrap exposes the underlying error so errors.Is / errors.As traverse it.
func (e *StoreError) Unwrap() error { return e.Err }

// LogValue implements [slog.LogValuer] so a *StoreError emits structured fields
// when logged via slog rather than a single string.
func (e *StoreError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("op", e.Op),
		slog.String("table", e.Table),
		slog.Any("cause", e.Err),
	)
}
