package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/atom"
)

// parseAtomEntries parses the raw Atom XML and returns a map from entry ID to
// atom.Entry for O(1) lookup when enriching universal Items.
func parseAtomEntries(body []byte) map[string]*atom.Entry {
	ap := atom.Parser{}
	atomFeed, err := ap.Parse(bytes.NewReader(body))
	if err != nil || atomFeed == nil {
		return nil
	}
	m := make(map[string]*atom.Entry, len(atomFeed.Entries))
	for _, e := range atomFeed.Entries {
		if e.ID != "" {
			m[e.ID] = e
		}
	}
	return m
}

func parseFetchedEntries(feedID int64, feedURL string, body []byte) ([]*Entry, error) {
	fp := gofeed.NewParser()
	parsed, err := fp.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: parse feed %s: %v", ErrInvalidFeed, feedURL, err)
	}

	var atomEntryByID map[string]*atom.Entry
	if strings.EqualFold(parsed.FeedType, "atom") {
		atomEntryByID = parseAtomEntries(body)
	}

	entries := make([]*Entry, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			continue
		}

		entry := buildEntry(feedID, guid, item, atomEntryByID[guid])
		entries = append(entries, entry)
	}

	return entries, nil
}

// buildEntry constructs an Entry from a universal gofeed.Item and an optional
// raw atom.Entry (for Atom-specific fields). The atomEntry may be nil for RSS
// or JSON feeds.
func buildEntry(feedID int64, guid string, item *gofeed.Item, atomEntry *atom.Entry) *Entry {
	e := &Entry{
		FeedID:       feedID,
		GUID:         guid,
		Title:        item.Title,
		Link:         item.Link,
		Summary:      item.Description,
		Content:      item.Content,
		Categories:   json.RawMessage("[]"),
		Enclosures:   json.RawMessage("[]"),
		Authors:      json.RawMessage("[]"),
		Links:        json.RawMessage("[]"),
		Contributors: json.RawMessage("[]"),
	}

	populateEntryTimes(e, item)
	populateEntryAuthorAndImage(e, item)
	populateEntryAuthors(e, item, atomEntry)
	populateEntryLinks(e, item, atomEntry)
	populateEntryCategories(e, item, atomEntry)
	populateEntryEnclosures(e, item)
	populateEntryContributors(e, atomEntry)
	populateEntryAtomMetadata(e, atomEntry)

	return e
}

func populateEntryTimes(e *Entry, item *gofeed.Item) {
	if item.PublishedParsed != nil {
		t := item.PublishedParsed.UTC()
		e.PublishedAt = &t
	}
	if item.UpdatedParsed != nil {
		t := item.UpdatedParsed.UTC()
		e.UpdatedAt = &t
	}
}

func populateEntryAuthorAndImage(e *Entry, item *gofeed.Item) {
	if len(item.Authors) > 0 && item.Authors[0] != nil {
		e.Author = item.Authors[0].Name
	} else if item.Author != nil {
		e.Author = item.Author.Name
	}
	if item.Image != nil {
		e.ImageURL = item.Image.URL
	}
}

func populateEntryAuthors(e *Entry, item *gofeed.Item, atomEntry *atom.Entry) {
	if atomEntry != nil && len(atomEntry.Authors) > 0 {
		persons := make([]EntryPerson, len(atomEntry.Authors))
		for i, a := range atomEntry.Authors {
			persons[i] = EntryPerson{Name: a.Name, Email: a.Email, URI: a.URI}
		}
		e.Authors, _ = json.Marshal(persons)
		return
	}
	if len(item.Authors) == 0 {
		return
	}
	persons := make([]EntryPerson, 0, len(item.Authors))
	for _, a := range item.Authors {
		if a != nil {
			persons = append(persons, EntryPerson{Name: a.Name, Email: a.Email})
		}
	}
	if len(persons) > 0 {
		e.Authors, _ = json.Marshal(persons)
	}
}

func populateEntryLinks(e *Entry, item *gofeed.Item, atomEntry *atom.Entry) {
	if atomEntry != nil && len(atomEntry.Links) > 0 {
		links := make([]EntryLink, len(atomEntry.Links))
		for i, l := range atomEntry.Links {
			links[i] = EntryLink{
				Href: l.Href, Rel: l.Rel, Type: l.Type,
				Hreflang: l.Hreflang, Title: l.Title, Length: l.Length,
			}
		}
		e.Links, _ = json.Marshal(links)
		return
	}
	if len(item.Links) == 0 {
		return
	}
	links := make([]EntryLink, len(item.Links))
	for i, href := range item.Links {
		links[i] = EntryLink{Href: href}
	}
	e.Links, _ = json.Marshal(links)
}

func populateEntryCategories(e *Entry, item *gofeed.Item, atomEntry *atom.Entry) {
	if atomEntry != nil && len(atomEntry.Categories) > 0 {
		cats := make([]EntryCategory, len(atomEntry.Categories))
		for i, c := range atomEntry.Categories {
			cats[i] = EntryCategory{Term: c.Term, Scheme: c.Scheme, Label: c.Label}
		}
		e.Categories, _ = json.Marshal(cats)
		return
	}
	if len(item.Categories) == 0 {
		return
	}
	cats := make([]EntryCategory, len(item.Categories))
	for i, c := range item.Categories {
		cats[i] = EntryCategory{Term: c}
	}
	e.Categories, _ = json.Marshal(cats)
}

func populateEntryEnclosures(e *Entry, item *gofeed.Item) {
	if len(item.Enclosures) == 0 {
		return
	}
	encs := make([]EntryEnclosure, len(item.Enclosures))
	for i, v := range item.Enclosures {
		encs[i] = EntryEnclosure{URL: v.URL, Length: v.Length, Type: v.Type}
	}
	e.Enclosures, _ = json.Marshal(encs)
}

func populateEntryContributors(e *Entry, atomEntry *atom.Entry) {
	if atomEntry == nil || len(atomEntry.Contributors) == 0 {
		return
	}
	persons := make([]EntryPerson, len(atomEntry.Contributors))
	for i, c := range atomEntry.Contributors {
		persons[i] = EntryPerson{Name: c.Name, Email: c.Email, URI: c.URI}
	}
	e.Contributors, _ = json.Marshal(persons)
}

func populateEntryAtomMetadata(e *Entry, atomEntry *atom.Entry) {
	if atomEntry == nil {
		return
	}
	e.Rights = atomEntry.Rights
	if atomEntry.Source == nil {
		return
	}
	src := EntrySource{
		Title:   atomEntry.Source.Title,
		ID:      atomEntry.Source.ID,
		Updated: atomEntry.Source.Updated,
	}
	e.Source, _ = json.Marshal(src)
}
