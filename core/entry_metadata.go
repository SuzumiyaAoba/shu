package core

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

// ParseCategories decodes Categories into a typed slice.
func (e *Entry) ParseCategories() ([]EntryCategory, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.categoriesParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.categories, e.metadataCache.categoriesErr
	}
	e.metadataCache.mu.RUnlock()

	values, err := parseRawSlice[EntryCategory](e.Categories, "categories")

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.categoriesParsed {
		e.metadataCache.categories = values
		e.metadataCache.categoriesErr = err
		e.metadataCache.categoriesParsed = true
	}
	return e.metadataCache.categories, e.metadataCache.categoriesErr
}

// ParseEnclosures decodes Enclosures into a typed slice.
func (e *Entry) ParseEnclosures() ([]EntryEnclosure, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.enclosuresParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.enclosures, e.metadataCache.enclosuresErr
	}
	e.metadataCache.mu.RUnlock()

	values, err := parseRawSlice[EntryEnclosure](e.Enclosures, "enclosures")

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.enclosuresParsed {
		e.metadataCache.enclosures = values
		e.metadataCache.enclosuresErr = err
		e.metadataCache.enclosuresParsed = true
	}
	return e.metadataCache.enclosures, e.metadataCache.enclosuresErr
}

// ParseAuthors decodes Authors into a typed slice.
func (e *Entry) ParseAuthors() ([]EntryPerson, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.authorsParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.authors, e.metadataCache.authorsErr
	}
	e.metadataCache.mu.RUnlock()

	values, err := parseRawSlice[EntryPerson](e.Authors, "authors")

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.authorsParsed {
		e.metadataCache.authors = values
		e.metadataCache.authorsErr = err
		e.metadataCache.authorsParsed = true
	}
	return e.metadataCache.authors, e.metadataCache.authorsErr
}

// ParseLinks decodes Links into a typed slice.
func (e *Entry) ParseLinks() ([]EntryLink, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.linksParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.links, e.metadataCache.linksErr
	}
	e.metadataCache.mu.RUnlock()

	values, err := parseRawSlice[EntryLink](e.Links, "links")

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.linksParsed {
		e.metadataCache.links = values
		e.metadataCache.linksErr = err
		e.metadataCache.linksParsed = true
	}
	return e.metadataCache.links, e.metadataCache.linksErr
}

// ParseContributors decodes Contributors into a typed slice.
func (e *Entry) ParseContributors() ([]EntryPerson, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.contributorsParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.contributors, e.metadataCache.contributorsErr
	}
	e.metadataCache.mu.RUnlock()

	values, err := parseRawSlice[EntryPerson](e.Contributors, "contributors")

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.contributorsParsed {
		e.metadataCache.contributors = values
		e.metadataCache.contributorsErr = err
		e.metadataCache.contributorsParsed = true
	}
	return e.metadataCache.contributors, e.metadataCache.contributorsErr
}

// ParseSource decodes Source into a typed object.
func (e *Entry) ParseSource() (*EntrySource, error) {
	e.metadataCache.mu.RLock()
	if e.metadataCache.sourceParsed {
		defer e.metadataCache.mu.RUnlock()
		return e.metadataCache.source, e.metadataCache.sourceErr
	}
	e.metadataCache.mu.RUnlock()

	var source *EntrySource
	var err error
	if !isEmptyRawMessage(e.Source) {
		parsed := new(EntrySource)
		if unmarshalErr := json.Unmarshal(e.Source, parsed); unmarshalErr != nil {
			err = fmt.Errorf("parse source: %w", unmarshalErr)
		} else {
			source = parsed
		}
	}

	e.metadataCache.mu.Lock()
	defer e.metadataCache.mu.Unlock()
	if !e.metadataCache.sourceParsed {
		e.metadataCache.source = source
		e.metadataCache.sourceErr = err
		e.metadataCache.sourceParsed = true
	}
	return e.metadataCache.source, e.metadataCache.sourceErr
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
