package core

import (
	"errors"
	"fmt"
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
