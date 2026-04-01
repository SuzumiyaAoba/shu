package core

import "errors"

var (
	// ErrFeedAlreadyExists indicates a feed with the same URL is already registered.
	ErrFeedAlreadyExists = errors.New("feed already exists")
	// ErrFeedNotFound indicates the requested feed does not exist.
	ErrFeedNotFound = errors.New("feed not found")
	// ErrEntryNotFound indicates the requested entry does not exist.
	ErrEntryNotFound = errors.New("entry not found")
)
