package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/atom"
)

const fetchWorkerCount = 10

type fetchedFeedDocument struct {
	body    []byte
	headers http.Header
}

type persistedFeedEntries struct {
	entries  []*Entry
	inserted int
}

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID, then
// stores any new entries that are not already in the database.
//
// For Atom feeds, the response body is parsed twice: once with gofeed's
// universal parser for common fields, and once with the Atom-specific parser
// to capture fields lost in translation (Contributors, Rights, Source,
// Author URIs, structured Categories, and full Link metadata).
//
// It returns a slice containing only the newly inserted entries.
func (s *Service) FetchFeed(ctx context.Context, feedID int64) ([]*Entry, error) {
	return s.FetchFeedWithObserver(ctx, feedID, nil)
}

// FetchFeedWithObserver downloads and parses a single feed while emitting
// structured progress events to observer.
func (s *Service) FetchFeedWithObserver(ctx context.Context, feedID int64, observer FetchObserver) ([]*Entry, error) {
	notifier := newFetchNotifier(observer)
	return s.fetchFeedByID(ctx, feedID, notifier)
}

func (s *Service) fetchFeedByID(ctx context.Context, feedID int64, notifier *fetchNotifier) ([]*Entry, error) {
	feed, err := s.store.GetFeed(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("get feed %d: %w", feedID, err)
	}

	return s.fetchFeed(ctx, feed, notifier)
}

func (s *Service) fetchFeed(ctx context.Context, feed *Feed, notifier *fetchNotifier) ([]*Entry, error) {
	emitFetchStarted(notifier, feed)

	if feed.Disabled {
		s.logger.Warn("feed disabled, skipping", "id", feed.ID, "title", feed.Title)
		emitFetchSkipped(notifier, feed, FetchSkipDisabled)
		return nil, nil
	}

	document, skipped, err := s.downloadFeedDocument(ctx, feed)
	if err != nil {
		emitFetchCompleted(notifier, feed, 0, err)
		return nil, err
	}
	if skipped {
		s.logger.Info("feed not modified", "id", feed.ID, "title", feed.Title)
		emitFetchSkipped(notifier, feed, FetchSkipNotModified)
		return nil, nil
	}

	result, err := s.persistFetchedFeed(ctx, feed, document)
	if err != nil {
		emitFetchCompleted(notifier, feed, 0, err)
		return nil, err
	}

	newEntries, count := s.resolveFetchedEntries(ctx, feed, result)
	emitFetchCompleted(notifier, feed, count, nil)
	return newEntries, nil
}

// fetchBody downloads the feed document at the given URL and returns the raw
// response body. This is used by AddFeed where no conditional GET is needed.
func (s *Service) fetchBody(ctx context.Context, url string) ([]byte, error) {
	body, _, err := s.fetchBodyConditional(ctx, url, "", "")
	return body, err
}

// fetchBodyConditional downloads the feed document with optional conditional
// GET headers (If-None-Match, If-Modified-Since). Returns nil body on 304.
func (s *Service) fetchBodyConditional(ctx context.Context, url, etag, lastModified string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := s.httpClient().Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}
	return body, resp.Header, nil
}

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

func (s *Service) downloadFeedDocument(ctx context.Context, feed *Feed) (*fetchedFeedDocument, bool, error) {
	body, headers, err := s.fetchBodyConditional(ctx, feed.URL, feed.ETag, feed.LastModified)
	if err != nil {
		return nil, false, s.handleFeedDownloadError(ctx, feed, err)
	}
	if body == nil {
		if err := s.markFeedFetched(ctx, feed.ID); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	}
	return &fetchedFeedDocument{body: body, headers: headers}, false, nil
}

func (s *Service) handleFeedDownloadError(ctx context.Context, feed *Feed, err error) error {
	fetchErr := fmt.Errorf("fetch feed %s: %w", feed.URL, err)
	if ctx.Err() != nil {
		return fetchErr
	}
	if recErr := s.store.RecordFeedError(ctx, feed.ID, err.Error()); recErr != nil {
		s.logger.Warn("failed to record feed error", "id", feed.ID, "error", recErr)
	}
	return fetchErr
}

func (s *Service) persistFetchedFeed(ctx context.Context, feed *Feed, document *fetchedFeedDocument) (*persistedFeedEntries, error) {
	s.storeConditionalHeaders(ctx, feed.ID, document.headers)

	entries, err := parseFetchedEntries(feed.ID, feed.URL, document.body)
	if err != nil {
		return nil, err
	}

	inserted, err := s.store.AddEntries(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("store entries: %w", err)
	}
	if err := s.markFeedFetched(ctx, feed.ID); err != nil {
		return nil, err
	}

	if err := s.store.ResetFeedError(ctx, feed.ID); err != nil {
		s.logger.Warn("failed to reset feed error", "id", feed.ID, "error", err)
	}

	s.logger.Info("feed fetched", "id", feed.ID, "title", feed.Title, "new_entries", inserted)
	return &persistedFeedEntries{entries: entries, inserted: inserted}, nil
}

func (s *Service) markFeedFetched(ctx context.Context, feedID int64) error {
	if err := s.store.UpdateFeedFetchedAt(ctx, feedID); err != nil {
		return fmt.Errorf("update fetched_at: %w", err)
	}
	return nil
}

func (s *Service) storeConditionalHeaders(ctx context.Context, feedID int64, headers http.Header) {
	if etag := headers.Get("ETag"); etag != "" || headers.Get("Last-Modified") != "" {
		if err := s.store.UpdateFeedCacheHeaders(ctx, feedID, headers.Get("ETag"), headers.Get("Last-Modified")); err != nil {
			s.logger.Warn("failed to update cache headers", "id", feedID, "error", err)
		}
	}
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

func (s *Service) resolveFetchedEntries(ctx context.Context, feed *Feed, result *persistedFeedEntries) ([]*Entry, int) {
	if result.inserted == 0 {
		return nil, 0
	}
	if result.inserted == len(result.entries) {
		return result.entries, len(result.entries)
	}

	feedID := feed.ID
	newEntries, err := s.store.ListEntries(ctx, EntryFilter{FeedID: &feedID, Limit: result.inserted})
	if err != nil {
		s.logger.Warn("failed to retrieve newly inserted entries", "id", feed.ID, "error", err)
		return result.entries[:result.inserted], result.inserted
	}
	return newEntries, len(newEntries)
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

func emitFetchStarted(notifier *fetchNotifier, feed *Feed) {
	notifier.emit(FetchEvent{
		Type:      FetchEventStarted,
		FeedID:    feed.ID,
		FeedTitle: feed.Title,
		FeedURL:   feed.URL,
	})
}

func emitFetchSkipped(notifier *fetchNotifier, feed *Feed, reason FetchSkipReason) {
	notifier.emit(FetchEvent{
		Type:       FetchEventSkipped,
		FeedID:     feed.ID,
		FeedTitle:  feed.Title,
		FeedURL:    feed.URL,
		SkipReason: reason,
	})
}

func emitFetchCompleted(notifier *fetchNotifier, feed *Feed, newEntries int, err error) {
	notifier.emit(FetchEvent{
		Type:       FetchEventCompleted,
		FeedID:     feed.ID,
		FeedTitle:  feed.Title,
		FeedURL:    feed.URL,
		NewEntries: newEntries,
		Err:        err,
	})
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

// FetchAll fetches every registered feed concurrently (up to 10 at a time) and
// returns the total number of new entries stored across all feeds.
//
// If an individual feed fails to fetch (network error, parse error, etc.), the
// error is logged and the method continues with the remaining feeds. This
// ensures that a single broken feed does not block updates for others.
//
// Feeds that have a per-feed interval set and were fetched more recently than
// that interval are skipped.
func (s *Service) FetchAll(ctx context.Context) (int, error) {
	return s.FetchAllWithObserver(ctx, nil)
}

// FetchAllWithObserver fetches all eligible feeds while emitting structured
// progress events to observer.
func (s *Service) FetchAllWithObserver(ctx context.Context, observer FetchObserver) (int, error) {
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("list feeds: %w", err)
	}

	notifier := newFetchNotifier(observer)
	toFetch := filterFeedsForFetch(feeds, notifier, time.Now())
	if len(toFetch) == 0 {
		return 0, nil
	}

	return s.fetchFeedsConcurrently(ctx, toFetch, notifier)
}

func filterFeedsForFetch(feeds []*Feed, notifier *fetchNotifier, now time.Time) []*Feed {
	toFetch := make([]*Feed, 0, len(feeds))
	for _, feed := range feeds {
		if shouldSkipFeedInterval(feed, now) {
			emitFetchSkipped(notifier, feed, FetchSkipInterval)
			continue
		}
		toFetch = append(toFetch, feed)
	}
	return toFetch
}

func shouldSkipFeedInterval(feed *Feed, now time.Time) bool {
	if feed.FetchIntervalSec <= 0 || feed.FetchedAt == nil {
		return false
	}
	return now.Sub(*feed.FetchedAt) < time.Duration(feed.FetchIntervalSec)*time.Second
}

func (s *Service) fetchFeedsConcurrently(ctx context.Context, feeds []*Feed, notifier *fetchNotifier) (int, error) {
	jobs := make(chan *Feed)
	workers := min(fetchWorkerCount, len(feeds))
	var total atomic.Int64
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case feed, ok := <-jobs:
				if !ok {
					return
				}

				entries, err := s.fetchFeed(ctx, feed, notifier)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					s.logger.Error("failed to fetch feed", "id", feed.ID, "url", feed.URL, "error", err)
					continue
				}
				total.Add(int64(len(entries)))
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	for _, feed := range feeds {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return int(total.Load()), ctx.Err()
		case jobs <- feed:
		}
	}

	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return int(total.Load()), err
	}

	return int(total.Load()), nil
}

// GetEntry retrieves a single entry by its primary key.
func (s *Service) GetEntry(ctx context.Context, id int64) (*Entry, error) {
	return s.store.GetEntry(ctx, id)
}

// ListEntries retrieves stored entries matching the given filter criteria.
// Results are ordered by fetched_at descending (newest first). It delegates
// directly to the store without additional business logic.
func (s *Service) ListEntries(ctx context.Context, filter EntryFilter) ([]*Entry, error) {
	return s.store.ListEntries(ctx, filter)
}
