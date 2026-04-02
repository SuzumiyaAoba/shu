package core

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// OPML represents an OPML 2.0 document for feed import/export.
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated,omitempty"`
}

type opmlBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

// OPMLOutline represents a single <outline> element. For feeds, Type is "rss"
// and XMLURL is the feed URL.
type OPMLOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []OPMLOutline `xml:"outline,omitempty"`
}

// ExportOPML generates an OPML document containing all registered feeds.
// Feeds are grouped by their tags; untagged feeds appear at the top level.
func (s *Service) ExportOPML(ctx context.Context) (*OPML, error) {
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}

	feedTags, err := s.store.ListFeedTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("list feed tags: %w", err)
	}

	taggedFeeds := make(map[string][]*Feed)
	tagNames := make([]string, 0)
	seenTags := make(map[string]bool)

	for _, f := range feeds {
		tags := feedTags[f.ID]
		if len(tags) == 0 {
			continue
		}
		for _, tag := range tags {
			taggedFeeds[tag.Name] = append(taggedFeeds[tag.Name], f)
			if !seenTags[tag.Name] {
				tagNames = append(tagNames, tag.Name)
				seenTags[tag.Name] = true
			}
		}
	}
	sort.Strings(tagNames)

	opml := &OPML{
		Version: "2.0",
		Head: opmlHead{
			Title:       "shu feed export",
			DateCreated: time.Now().UTC().Format(time.RFC1123Z),
		},
	}

	// Untagged feeds at top level.
	for _, f := range feeds {
		if len(feedTags[f.ID]) == 0 {
			opml.Body.Outlines = append(opml.Body.Outlines, feedToOutline(f))
		}
	}

	// Tagged feeds grouped under category outlines.
	for _, tagName := range tagNames {
		group := OPMLOutline{Text: tagName, Title: tagName}
		for _, f := range taggedFeeds[tagName] {
			group.Outlines = append(group.Outlines, feedToOutline(f))
		}
		if len(group.Outlines) > 0 {
			opml.Body.Outlines = append(opml.Body.Outlines, group)
		}
	}

	return opml, nil
}

func feedToOutline(f *Feed) OPMLOutline {
	return OPMLOutline{
		Text:    f.Title,
		Title:   f.Title,
		Type:    "rss",
		XMLURL:  f.URL,
		HTMLURL: f.SiteURL,
	}
}

// ImportOPML reads an OPML document and adds all feeds found in it.
// Nested outlines (categories) are imported as cumulative tags from root to
// leaf. Duplicate feeds are reused; other add failures are returned.
func (s *Service) ImportOPML(ctx context.Context, r io.Reader) (int, error) {
	var opml OPML
	if err := xml.NewDecoder(r).Decode(&opml); err != nil {
		return 0, fmt.Errorf("%w: parse OPML: %v", ErrInvalidOPML, err)
	}

	added := 0
	for _, outline := range opml.Body.Outlines {
		n, err := s.importOutline(ctx, outline, nil)
		if err != nil {
			return added, err
		}
		added += n
	}
	return added, nil
}

func (s *Service) importOutline(ctx context.Context, outline OPMLOutline, parentTags []string) (int, error) {
	// If this outline has children, treat it as a category.
	if len(outline.Outlines) > 0 {
		tag := outline.Text
		if tag == "" {
			tag = outline.Title
		}
		tags := parentTags
		if tag != "" {
			tags = append(append([]string{}, parentTags...), tag)
		}
		added := 0
		for _, child := range outline.Outlines {
			n, err := s.importOutline(ctx, child, tags)
			if err != nil {
				return added, err
			}
			added += n
		}
		return added, nil
	}

	// Leaf outline — treat as a feed.
	if outline.XMLURL == "" {
		return 0, nil
	}

	title := outline.Title
	if title == "" {
		title = outline.Text
	}

	feed, err := s.AddFeed(ctx, outline.XMLURL, title)
	if err != nil {
		if errors.Is(err, ErrFeedAlreadyExists) {
			s.logger.Info("reuse duplicate OPML feed", "url", outline.XMLURL)
			existingFeed, getErr := s.store.GetFeedByURL(ctx, outline.XMLURL)
			if getErr != nil {
				return 0, fmt.Errorf("get duplicate OPML feed %s: %w", outline.XMLURL, getErr)
			}
			if err := s.applyTags(ctx, existingFeed.ID, outline.XMLURL, parentTags); err != nil {
				return 0, err
			}
			return 0, nil
		}
		return 0, fmt.Errorf("add OPML feed %s: %w", outline.XMLURL, err)
	}

	if err := s.applyTags(ctx, feed.ID, outline.XMLURL, parentTags); err != nil {
		return 0, err
	}

	return 1, nil
}

func (s *Service) applyTags(ctx context.Context, feedID int64, feedURL string, tags []string) error {
	for _, tag := range tags {
		if err := s.AddTag(ctx, feedID, tag); err != nil {
			return fmt.Errorf("%w: tag OPML feed %s with %q: %v", ErrTagApplyFailed, feedURL, tag, err)
		}
	}
	return nil
}
