package core

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
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

type opmlFeedAdder interface {
	AddFeed(ctx context.Context, url string, titleOverride string) (*Feed, error)
	AddFeedDirect(ctx context.Context, feed *Feed) error
}

type opmlTagAdder interface {
	AddTag(ctx context.Context, feedID int64, tagName string) error
}

// OPMLHandler owns OPML import/export workflows.
type OPMLHandler struct {
	feedStore FeedStore
	tagStore  TagStore
	feeds     opmlFeedAdder
	tags      opmlTagAdder
	logger    *slog.Logger
}

// NewOPMLHandler creates an OPML domain service.
func NewOPMLHandler(feedStore FeedStore, tagStore TagStore, feeds opmlFeedAdder, tags opmlTagAdder, logger *slog.Logger) *OPMLHandler {
	return &OPMLHandler{
		feedStore: feedStore,
		tagStore:  tagStore,
		feeds:     feeds,
		tags:      tags,
		logger:    normalizeLogger(logger),
	}
}

// ExportOPML generates an OPML document containing all registered feeds.
// Feeds are grouped by their tags; untagged feeds appear at the top level.
func (h *OPMLHandler) ExportOPML(ctx context.Context) (*OPML, error) {
	feeds, err := h.feedStore.ListFeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}

	feedTags, err := h.tagStore.ListFeedTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("list feed tags: %w", err)
	}

	taggedFeeds := make(map[string][]*Feed)
	for _, f := range feeds {
		for _, tag := range feedTags[f.ID] {
			taggedFeeds[tag.Name] = append(taggedFeeds[tag.Name], f)
		}
	}
	tagNames := slices.Sorted(maps.Keys(taggedFeeds))

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
func (h *OPMLHandler) ImportOPML(ctx context.Context, r io.Reader) (int, error) {
	opml, err := decodeOPML(r)
	if err != nil {
		return 0, err
	}

	importer := newOPMLImporter(h, nil)
	return importer.importAll(ctx, opml.Body.Outlines)
}

// ImportOPMLDetailed imports an OPML document and returns a detailed summary of
// additions, reused feeds, tag applications, and non-fatal issues.
func (h *OPMLHandler) ImportOPMLDetailed(ctx context.Context, r io.Reader) (*OPMLImportResult, error) {
	opml, err := decodeOPML(r)
	if err != nil {
		return &OPMLImportResult{}, err
	}

	result := &OPMLImportResult{}
	importer := newOPMLImporter(h, result)
	_, err = importer.importAll(ctx, opml.Body.Outlines)
	return result, err
}

func decodeOPML(r io.Reader) (*OPML, error) {
	var opml OPML
	if err := xml.NewDecoder(r).Decode(&opml); err != nil {
		return nil, fmt.Errorf("%w: parse OPML: %v", ErrInvalidOPML, err)
	}
	return &opml, nil
}

type opmlImporter struct {
	handler *OPMLHandler
	result  *OPMLImportResult
}

func newOPMLImporter(handler *OPMLHandler, result *OPMLImportResult) *opmlImporter {
	return &opmlImporter{handler: handler, result: result}
}

func (i *opmlImporter) importAll(ctx context.Context, outlines []OPMLOutline) (int, error) {
	added := 0
	for _, outline := range outlines {
		n, err := i.importOutline(ctx, outline, nil)
		if err != nil {
			return added, err
		}
		added += n
	}
	return added, nil
}

func (i *opmlImporter) importOutline(ctx context.Context, outline OPMLOutline, parentTags []string) (int, error) {
	if len(outline.Outlines) > 0 {
		tag := outline.Text
		if tag == "" {
			tag = outline.Title
		}
		tags := parentTags
		if tag != "" {
			tags = append(slices.Clone(parentTags), tag)
		}
		added := 0
		for _, child := range outline.Outlines {
			n, err := i.importOutline(ctx, child, tags)
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

	feed, added, err := i.ensureFeed(ctx, outline.XMLURL, title)
	if err != nil {
		return 0, err
	}
	if feed == nil {
		return 0, nil
	}

	if err := i.applyTags(ctx, feed.ID, outline.XMLURL, parentTags); err != nil {
		return added, err
	}
	return added, nil
}

func (i *opmlImporter) ensureFeed(ctx context.Context, url, title string) (*Feed, int, error) {
	feed := &Feed{URL: url, Title: title}
	err := i.handler.feeds.AddFeedDirect(ctx, feed)
	if err == nil {
		if i.result != nil {
			i.result.AddedCount++
		}
		return feed, 1, nil
	}

	if errors.Is(err, ErrFeedAlreadyExists) {
		i.handler.logger.Info("reuse duplicate OPML feed", "url", url)
		if i.result != nil {
			i.result.ReusedCount++
		}
		existingFeed, getErr := i.handler.feedStore.GetFeedByURL(ctx, url)
		if getErr != nil {
			return nil, 0, fmt.Errorf("get duplicate OPML feed %s: %w", url, getErr)
		}
		return existingFeed, 0, nil
	}

	if i.result != nil {
		i.result.Issues = append(i.result.Issues, OPMLImportIssue{
			URL:   url,
			Error: err.Error(),
		})
		return nil, 0, nil
	}

	return nil, 0, fmt.Errorf("add OPML feed %s: %w", url, err)
}

func (i *opmlImporter) applyTags(ctx context.Context, feedID int64, feedURL string, tags []string) error {
	for _, tag := range tags {
		if err := i.handler.tags.AddTag(ctx, feedID, tag); err != nil {
			tagErr := fmt.Errorf("%w: tag OPML feed %s with %q: %v", ErrTagApplyFailed, feedURL, tag, err)
			if i.result != nil {
				i.result.Issues = append(i.result.Issues, OPMLImportIssue{
					URL:   feedURL,
					Tag:   tag,
					Error: tagErr.Error(),
				})
				return nil
			}
			return tagErr
		}
		if i.result != nil {
			i.result.TaggedCount++
		}
	}
	return nil
}

func (s *Service) ExportOPML(ctx context.Context) (*OPML, error) {
	return s.opml.ExportOPML(ctx)
}

func (s *Service) ImportOPML(ctx context.Context, r io.Reader) (int, error) {
	return s.opml.ImportOPML(ctx, r)
}

func (s *Service) ImportOPMLDetailed(ctx context.Context, r io.Reader) (*OPMLImportResult, error) {
	return s.opml.ImportOPMLDetailed(ctx, r)
}
