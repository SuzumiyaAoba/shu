package core

import "errors"

var (
	// ErrFeedNotFound indicates the requested feed does not exist.
	ErrFeedNotFound = errors.New("feed not found")
	// ErrEntryNotFound indicates the requested entry does not exist.
	ErrEntryNotFound = errors.New("entry not found")
)
