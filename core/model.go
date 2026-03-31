package core

import (
	"encoding/json"
	"time"
)

// Feed represents a registered RSS/Atom feed source.
// A feed is identified by its unique URL and contains metadata extracted from
// the feed document (title, site URL) as well as bookkeeping timestamps.
type Feed struct {
	// ID is the auto-generated database primary key.
	ID int64 `json:"id"`
	// URL is the feed endpoint (e.g. "https://example.com/feed.xml").
	// It is unique across all registered feeds.
	URL string `json:"url"`
	// Title is the human-readable name of the feed, typically extracted from
	// the <title> element in the feed document. It can be overridden by the
	// user when adding the feed.
	Title string `json:"title"`
	// SiteURL is the URL of the website associated with the feed, extracted
	// from the <link> element in the feed document.
	SiteURL string `json:"site_url"`
	// AddedAt records when the feed was first registered.
	AddedAt time.Time `json:"added_at"`
	// FetchedAt records the last time the feed was successfully fetched.
	// It is nil if the feed has never been fetched after registration.
	FetchedAt *time.Time `json:"fetched_at"`
	// Description is the feed's tagline or description text, extracted from
	// the <description> (RSS) or <subtitle> (Atom) element.
	Description string `json:"description"`
	// Language is the feed's declared language code (e.g. "en", "ja"),
	// extracted from the <language> element.
	Language string `json:"language"`
	// ImageURL is the URL of the feed's logo or icon image, extracted from
	// the <image><url> element.
	ImageURL string `json:"image_url"`
	// FeedType indicates the detected feed format as reported by gofeed:
	// "rss", "atom", or "json".
	FeedType string `json:"feed_type"`
	// ETag is the HTTP ETag header value from the last successful fetch.
	// Used for conditional GET requests to avoid re-downloading unchanged feeds.
	ETag string `json:"e_tag"`
	// LastModified is the HTTP Last-Modified header value from the last
	// successful fetch. Used together with ETag for conditional GET requests.
	LastModified string `json:"last_modified"`
	// ErrorCount is the number of consecutive fetch failures.
	ErrorCount int `json:"error_count"`
	// LastError is the error message from the most recent failed fetch.
	LastError string `json:"last_error"`
	// Disabled indicates the feed is temporarily disabled due to repeated failures.
	Disabled bool `json:"disabled"`
	// FetchIntervalSec is a per-feed override for the fetch interval in seconds.
	// 0 means use the global default.
	FetchIntervalSec int `json:"fetch_interval_sec"`
}

// Entry represents a single article or item from an RSS/Atom feed.
// Entries are deduplicated by the combination of FeedID and GUID, so fetching
// the same feed multiple times will not create duplicate entries.
type Entry struct {
	// ID is the auto-generated database primary key.
	ID int64 `json:"id"`
	// FeedID is the foreign key referencing the parent Feed.
	// Entries are cascade-deleted when their parent feed is removed.
	FeedID int64 `json:"feed_id"`
	// GUID is the globally unique identifier for this entry within its feed.
	// It comes from the <guid> element in RSS or <id> in Atom. If neither is
	// present in the source feed, the entry's Link is used as a fallback.
	GUID string `json:"guid"`
	// Title is the headline of the entry.
	Title string `json:"title"`
	// Link is the permalink URL to the full article.
	Link string `json:"link"`
	// Summary is a short description or excerpt of the entry content,
	// taken from the <description> (RSS) or <summary> (Atom) element.
	Summary string `json:"summary"`
	// PublishedAt is the publication date as declared by the feed.
	// It is nil if the source feed does not provide a publication date.
	PublishedAt *time.Time `json:"published_at"`
	// FetchedAt records when this entry was first stored in the database.
	FetchedAt time.Time `json:"fetched_at"`
	// Content is the full HTML content of the entry, from <content:encoded>
	// (RSS) or <content> (Atom). It may be empty if the feed only provides
	// a summary.
	Content string `json:"content"`
	// Author is the entry author's display name, taken from the first element
	// of the source feed's authors list. Empty if no author is specified.
	Author string `json:"author"`
	// ImageURL is the URL of the entry's featured image or thumbnail.
	ImageURL string `json:"image_url"`
	// Categories is a JSON-encoded array of category objects.
	// Each element has the shape {"term":"...","scheme":"...","label":"..."}.
	// For RSS feeds, term is the category string and scheme/label are empty.
	// Defaults to "[]" when no categories are present.
	Categories json.RawMessage `json:"categories"`
	// UpdatedAt is when the entry was last modified by the source feed.
	// It is nil if the source does not provide an update timestamp.
	UpdatedAt *time.Time `json:"updated_at"`
	// Enclosures is a JSON-encoded array of media attachment objects.
	// Each element has the shape {"url":"...","length":"...","type":"..."}.
	// Defaults to "[]" when no enclosures are present.
	Enclosures json.RawMessage `json:"enclosures"`
	// Authors is a JSON-encoded array of person objects with full metadata.
	// Each element has the shape {"name":"...","email":"...","uri":"..."}.
	// The Author field (above) is kept for convenience as the first author's name.
	// Defaults to "[]" when no authors are specified.
	Authors json.RawMessage `json:"authors"`
	// Links is a JSON-encoded array of link objects with full Atom metadata.
	// Each element has the shape {"href":"...","rel":"...","type":"...","hreflang":"...","title":"...","length":"..."}.
	// The Link field (above) is kept for convenience as the primary permalink.
	// Defaults to "[]".
	Links json.RawMessage `json:"links"`
	// Contributors is a JSON-encoded array of person objects (Atom only).
	// Each element has the shape {"name":"...","email":"...","uri":"..."}.
	// Defaults to "[]". Only populated for Atom feeds.
	Contributors json.RawMessage `json:"contributors"`
	// Rights is the copyright or license text from the Atom <rights> element.
	// Empty for RSS feeds or when not specified.
	Rights string `json:"rights"`
	// Source is a JSON-encoded object representing the Atom <source> element,
	// with the shape {"title":"...","id":"...","updated":"..."}.
	// Empty string when not present. Only populated for Atom feeds.
	Source json.RawMessage `json:"source"`
	// ReadAt records when the entry was marked as read by the user.
	// It is nil if the entry has not been read yet.
	ReadAt *time.Time `json:"read_at"`
	// StarredAt records when the entry was bookmarked/starred.
	// It is nil if the entry has not been starred.
	StarredAt *time.Time `json:"starred_at"`
}

// EntryFilter specifies criteria for querying stored entries.
type EntryFilter struct {
	// FeedID, when non-nil, restricts results to entries belonging to the
	// specified feed.
	FeedID *int64 `json:"feed_id"`
	// Limit caps the number of entries returned. A value of 0 means no limit.
	Limit int `json:"limit"`
	// Offset skips the first N entries in the result set, enabling pagination
	// when combined with Limit.
	Offset int `json:"offset"`
	// UnreadOnly, when true, restricts results to entries that have not been
	// marked as read (read_at IS NULL).
	UnreadOnly bool `json:"unread_only"`
	// Tag, when non-empty, restricts results to entries from feeds that have
	// the specified tag.
	Tag string `json:"tag"`
	// StarredOnly, when true, restricts results to starred entries.
	StarredOnly bool `json:"starred_only"`
}

// FeedUpdate holds the mutable fields for updating a feed.
// Nil pointer fields are left unchanged.
type FeedUpdate struct {
	Title *string `json:"title"`
	URL   *string `json:"url"`
}

// Tag represents a user-defined label for organizing feeds.
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// FeedStats holds aggregate statistics for a single feed.
type FeedStats struct {
	FeedID       int64      `json:"feed_id"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	TotalCount   int        `json:"total_count"`
	UnreadCount  int        `json:"unread_count"`
	StarredCount int        `json:"starred_count"`
	FetchedAt    *time.Time `json:"fetched_at"`
	ErrorCount   int        `json:"error_count"`
	LastError    string     `json:"last_error"`
	Disabled     bool       `json:"disabled"`
}
