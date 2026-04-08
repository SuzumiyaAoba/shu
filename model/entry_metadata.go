package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// EntryPerson is the structured representation of an author or contributor.
type EntryPerson struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URI   string `json:"uri"`
}

// EntryLink is the structured representation of an entry link.
type EntryLink struct {
	Href     string `json:"href"`
	Rel      string `json:"rel"`
	Type     string `json:"type"`
	Hreflang string `json:"hreflang"`
	Title    string `json:"title"`
	Length   string `json:"length"`
}

// EntryCategory is the structured representation of an entry category.
type EntryCategory struct {
	Term   string `json:"term"`
	Scheme string `json:"scheme"`
	Label  string `json:"label"`
}

// EntryEnclosure is the structured representation of an entry enclosure.
type EntryEnclosure struct {
	URL    string `json:"url"`
	Length string `json:"length"`
	Type   string `json:"type"`
}

// EntrySource is the structured representation of an Atom source element.
type EntrySource struct {
	Title   string `json:"title"`
	ID      string `json:"id"`
	Updated string `json:"updated"`
}

// ParseCategories decodes the entry's Categories JSON into a typed slice.
func ParseCategories(e *Entry) ([]EntryCategory, error) {
	return ParseRawSlice[EntryCategory](e.Categories, "categories")
}

// ParseEnclosures decodes the entry's Enclosures JSON into a typed slice.
func ParseEnclosures(e *Entry) ([]EntryEnclosure, error) {
	return ParseRawSlice[EntryEnclosure](e.Enclosures, "enclosures")
}

// ParseAuthors decodes the entry's Authors JSON into a typed slice.
func ParseAuthors(e *Entry) ([]EntryPerson, error) {
	return ParseRawSlice[EntryPerson](e.Authors, "authors")
}

// ParseLinks decodes the entry's Links JSON into a typed slice.
func ParseLinks(e *Entry) ([]EntryLink, error) {
	return ParseRawSlice[EntryLink](e.Links, "links")
}

// ParseContributors decodes the entry's Contributors JSON into a typed slice.
func ParseContributors(e *Entry) ([]EntryPerson, error) {
	return ParseRawSlice[EntryPerson](e.Contributors, "contributors")
}

// ParseSource decodes the entry's Source JSON into a typed object.
func ParseSource(e *Entry) (*EntrySource, error) {
	if IsEmptyRawMessage(e.Source) {
		return nil, nil
	}
	var source EntrySource
	if err := json.Unmarshal(e.Source, &source); err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}
	return &source, nil
}

// ParseRawSlice parses a JSON-encoded array into a typed slice.
func ParseRawSlice[T any](data json.RawMessage, field string) ([]T, error) {
	if IsEmptyRawMessage(data) {
		return []T{}, nil
	}

	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	if values == nil {
		return []T{}, nil
	}
	return values, nil
}

// IsEmptyRawMessage reports whether data is empty, whitespace-only, or "null".
func IsEmptyRawMessage(data json.RawMessage) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
