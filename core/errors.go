package core

import "errors"

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
