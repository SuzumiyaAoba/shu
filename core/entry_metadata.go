package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
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

// cachedValue is a generic container for lazily-parsed, thread-safe cached data.
type cachedValue[T any] struct {
	mu     sync.RWMutex
	parsed bool
	value  T
	err    error
}

// getOrParse returns the cached value, computing it via parse on first access.
func (c *cachedValue[T]) getOrParse(parse func() (T, error)) (T, error) {
	c.mu.RLock()
	if c.parsed {
		defer c.mu.RUnlock()
		return c.value, c.err
	}
	c.mu.RUnlock()

	value, err := parse()

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.parsed {
		c.value = value
		c.err = err
		c.parsed = true
	}
	return c.value, c.err
}

// ParseCategories decodes Categories into a typed slice.
func (e *Entry) ParseCategories() ([]EntryCategory, error) {
	return e.metadataCache.categories.getOrParse(func() ([]EntryCategory, error) {
		return parseRawSlice[EntryCategory](e.Categories, "categories")
	})
}

// ParseEnclosures decodes Enclosures into a typed slice.
func (e *Entry) ParseEnclosures() ([]EntryEnclosure, error) {
	return e.metadataCache.enclosures.getOrParse(func() ([]EntryEnclosure, error) {
		return parseRawSlice[EntryEnclosure](e.Enclosures, "enclosures")
	})
}

// ParseAuthors decodes Authors into a typed slice.
func (e *Entry) ParseAuthors() ([]EntryPerson, error) {
	return e.metadataCache.authors.getOrParse(func() ([]EntryPerson, error) {
		return parseRawSlice[EntryPerson](e.Authors, "authors")
	})
}

// ParseLinks decodes Links into a typed slice.
func (e *Entry) ParseLinks() ([]EntryLink, error) {
	return e.metadataCache.links.getOrParse(func() ([]EntryLink, error) {
		return parseRawSlice[EntryLink](e.Links, "links")
	})
}

// ParseContributors decodes Contributors into a typed slice.
func (e *Entry) ParseContributors() ([]EntryPerson, error) {
	return e.metadataCache.contributors.getOrParse(func() ([]EntryPerson, error) {
		return parseRawSlice[EntryPerson](e.Contributors, "contributors")
	})
}

// ParseSource decodes Source into a typed object.
func (e *Entry) ParseSource() (*EntrySource, error) {
	return e.metadataCache.source.getOrParse(func() (*EntrySource, error) {
		if isEmptyRawMessage(e.Source) {
			return nil, nil
		}
		var source EntrySource
		if err := json.Unmarshal(e.Source, &source); err != nil {
			return nil, fmt.Errorf("parse source: %w", err)
		}
		return &source, nil
	})
}

func parseRawSlice[T any](data json.RawMessage, field string) ([]T, error) {
	if isEmptyRawMessage(data) {
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

func isEmptyRawMessage(data json.RawMessage) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
