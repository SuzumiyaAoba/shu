package core

import "time"

// Feed represents a registered RSS/Atom feed source.
// A feed is identified by its unique URL and contains metadata extracted from
// the feed document (title, site URL) as well as bookkeeping timestamps.
type Feed struct {
	// ID is the auto-generated database primary key.
	ID int64
	// URL is the feed endpoint (e.g. "https://example.com/feed.xml").
	// It is unique across all registered feeds.
	URL string
	// Title is the human-readable name of the feed, typically extracted from
	// the <title> element in the feed document. It can be overridden by the
	// user when adding the feed.
	Title string
	// SiteURL is the URL of the website associated with the feed, extracted
	// from the <link> element in the feed document.
	SiteURL string
	// AddedAt records when the feed was first registered.
	AddedAt time.Time
	// FetchedAt records the last time the feed was successfully fetched.
	// It is nil if the feed has never been fetched after registration.
	FetchedAt *time.Time
	// Description is the feed's tagline or description text, extracted from
	// the <description> (RSS) or <subtitle> (Atom) element.
	Description string
	// Language is the feed's declared language code (e.g. "en", "ja"),
	// extracted from the <language> element.
	Language string
	// ImageURL is the URL of the feed's logo or icon image, extracted from
	// the <image><url> element.
	ImageURL string
	// FeedType indicates the detected feed format as reported by gofeed:
	// "rss", "atom", or "json".
	FeedType string
}

// Entry represents a single article or item from an RSS/Atom feed.
// Entries are deduplicated by the combination of FeedID and GUID, so fetching
// the same feed multiple times will not create duplicate entries.
type Entry struct {
	// ID is the auto-generated database primary key.
	ID int64
	// FeedID is the foreign key referencing the parent Feed.
	// Entries are cascade-deleted when their parent feed is removed.
	FeedID int64
	// GUID is the globally unique identifier for this entry within its feed.
	// It comes from the <guid> element in RSS or <id> in Atom. If neither is
	// present in the source feed, the entry's Link is used as a fallback.
	GUID string
	// Title is the headline of the entry.
	Title string
	// Link is the permalink URL to the full article.
	Link string
	// Summary is a short description or excerpt of the entry content,
	// taken from the <description> (RSS) or <summary> (Atom) element.
	Summary string
	// PublishedAt is the publication date as declared by the feed.
	// It is nil if the source feed does not provide a publication date.
	PublishedAt *time.Time
	// FetchedAt records when this entry was first stored in the database.
	FetchedAt time.Time
	// Content is the full HTML content of the entry, from <content:encoded>
	// (RSS) or <content> (Atom). It may be empty if the feed only provides
	// a summary.
	Content string
	// Author is the entry author's display name, taken from the first element
	// of the source feed's authors list. Empty if no author is specified.
	Author string
	// ImageURL is the URL of the entry's featured image or thumbnail.
	ImageURL string
	// Categories is a JSON-encoded array of tag/category strings
	// (e.g. `["go","rss"]`). Defaults to "[]" when no categories are present.
	Categories string
	// UpdatedAt is when the entry was last modified by the source feed.
	// It is nil if the source does not provide an update timestamp.
	UpdatedAt *time.Time
	// Enclosures is a JSON-encoded array of media attachment objects.
	// Each element has the shape {"url":"...","length":"...","type":"..."}.
	// Defaults to "[]" when no enclosures are present.
	Enclosures string
}

// EntryFilter specifies criteria for querying stored entries.
type EntryFilter struct {
	// FeedID, when non-nil, restricts results to entries belonging to the
	// specified feed.
	FeedID *int64
	// Limit caps the number of entries returned. A value of 0 means no limit.
	Limit int
	// Offset skips the first N entries in the result set, enabling pagination
	// when combined with Limit.
	Offset int
}
