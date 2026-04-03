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

// OPMLImportIssue records a non-fatal issue encountered during import.
type OPMLImportIssue struct {
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
	Error string `json:"error"`
}

// OPMLImportResult summarizes an import run.
type OPMLImportResult struct {
	AddedCount  int               `json:"added_count"`
	ReusedCount int               `json:"reused_count"`
	TaggedCount int               `json:"tagged_count"`
	Issues      []OPMLImportIssue `json:"issues,omitempty"`
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
	opml, err := decodeOPML(r)
	if err != nil {
		return 0, err
	}

	added := 0
	for _, outline := range opml.Body.Outlines {
		n, err := s.importOutlineStrict(ctx, outline, nil)
		if err != nil {
			return added, err
		}
		added += n
	}
	return added, nil
}

// ImportOPMLDetailed imports an OPML document and returns a detailed summary of
// additions, reused feeds, tag applications, and non-fatal issues.
func (s *Service) ImportOPMLDetailed(ctx context.Context, r io.Reader) (*OPMLImportResult, error) {
	opml, err := decodeOPML(r)
	if err != nil {
		return &OPMLImportResult{}, err
	}

	result := &OPMLImportResult{}
	for _, outline := range opml.Body.Outlines {
		if err := s.importOutline(ctx, outline, nil, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func decodeOPML(r io.Reader) (*OPML, error) {
	var opml OPML
	if err := xml.NewDecoder(r).Decode(&opml); err != nil {
		return nil, fmt.Errorf("%w: parse OPML: %v", ErrInvalidOPML, err)
	}
	return &opml, nil
}

func (s *Service) importOutlineStrict(ctx context.Context, outline OPMLOutline, parentTags []string) (int, error) {
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
			n, err := s.importOutlineStrict(ctx, child, tags)
			if err != nil {
				return added, err
			}
			added += n
		}
		return added, nil
	}

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
			if err := s.applyTagsStrict(ctx, existingFeed.ID, outline.XMLURL, parentTags); err != nil {
				return 0, err
			}
			return 0, nil
		}
		return 0, fmt.Errorf("add OPML feed %s: %w", outline.XMLURL, err)
	}

	if err := s.applyTagsStrict(ctx, feed.ID, outline.XMLURL, parentTags); err != nil {
		return 0, err
	}
	return 1, nil
}

func (s *Service) importOutline(ctx context.Context, outline OPMLOutline, parentTags []string, result *OPMLImportResult) error {
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
		for _, child := range outline.Outlines {
			if err := s.importOutline(ctx, child, tags, result); err != nil {
				return err
			}
		}
		return nil
	}

	// Leaf outline — treat as a feed.
	if outline.XMLURL == "" {
		return nil
	}

	title := outline.Title
	if title == "" {
		title = outline.Text
	}

	feed, err := s.AddFeed(ctx, outline.XMLURL, title)
	if err != nil {
		if errors.Is(err, ErrFeedAlreadyExists) {
			s.logger.Info("reuse duplicate OPML feed", "url", outline.XMLURL)
			result.ReusedCount++
			existingFeed, getErr := s.store.GetFeedByURL(ctx, outline.XMLURL)
			if getErr != nil {
				return fmt.Errorf("get duplicate OPML feed %s: %w", outline.XMLURL, getErr)
			}
			if tagErr := s.applyTags(ctx, existingFeed.ID, outline.XMLURL, parentTags, result); tagErr != nil {
				result.Issues = append(result.Issues, OPMLImportIssue{
					URL:   outline.XMLURL,
					Error: tagErr.Error(),
				})
			}
			return nil
		}
		result.Issues = append(result.Issues, OPMLImportIssue{
			URL:   outline.XMLURL,
			Error: err.Error(),
		})
		return nil
	}
	result.AddedCount++

	if tagErr := s.applyTags(ctx, feed.ID, outline.XMLURL, parentTags, result); tagErr != nil {
		result.Issues = append(result.Issues, OPMLImportIssue{
			URL:   outline.XMLURL,
			Error: tagErr.Error(),
		})
	}
	return nil
}

func (s *Service) applyTags(ctx context.Context, feedID int64, feedURL string, tags []string, result *OPMLImportResult) error {
	for _, tag := range tags {
		if err := s.AddTag(ctx, feedID, tag); err != nil {
			return fmt.Errorf("%w: tag OPML feed %s with %q: %v", ErrTagApplyFailed, feedURL, tag, err)
		}
		result.TaggedCount++
	}
	return nil
}

func (s *Service) applyTagsStrict(ctx context.Context, feedID int64, feedURL string, tags []string) error {
	for _, tag := range tags {
		if err := s.AddTag(ctx, feedID, tag); err != nil {
			return fmt.Errorf("%w: tag OPML feed %s with %q: %v", ErrTagApplyFailed, feedURL, tag, err)
		}
	}
	return nil
}
