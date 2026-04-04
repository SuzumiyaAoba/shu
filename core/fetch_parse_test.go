package core

import (
	"errors"
	"strings"
	"testing"
)

const parseTestRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test Blog</title>
    <item>
      <title>Post 1</title>
      <link>https://example.com/post-1</link>
      <guid>post-1</guid>
      <description>First post</description>
      <content:encoded><![CDATA[<p>Full content of post 1</p>]]></content:encoded>
      <author>alice@example.com (Alice)</author>
      <category>Go</category>
      <enclosure url="https://example.com/ep1.mp3" length="12345" type="audio/mpeg"/>
    </item>
    <item>
      <title>Post 2</title>
      <link>https://example.com/post-2</link>
      <guid>post-2</guid>
      <description>Second post</description>
    </item>
  </channel>
</rss>`

const parseTestAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test Blog</title>
  <id>urn:uuid:feed-1</id>
  <updated>2026-03-01T12:00:00Z</updated>
  <entry>
    <title>Atom Post 1</title>
    <id>urn:uuid:entry-1</id>
    <link href="https://example.com/atom-post-1" rel="alternate" type="text/html" hreflang="en"/>
    <updated>2026-03-01T12:00:00Z</updated>
    <published>2026-02-28T10:00:00Z</published>
    <summary>Atom summary</summary>
    <content type="html">&lt;p&gt;Full Atom content&lt;/p&gt;</content>
    <author>
      <name>Alice</name>
      <email>alice@example.com</email>
      <uri>https://alice.example.com</uri>
    </author>
    <contributor>
      <name>Charlie</name>
      <email>charlie@example.com</email>
      <uri>https://charlie.example.com</uri>
    </contributor>
    <category term="golang" scheme="https://example.com/tags" label="Go Language"/>
    <rights>CC BY 4.0</rights>
    <source>
      <title>Original Source</title>
      <id>urn:uuid:source-1</id>
      <updated>2026-01-01T00:00:00Z</updated>
    </source>
  </entry>
</feed>`

func TestParseFetchedEntriesRSS(t *testing.T) {
	entries, err := parseFetchedEntries(42, "https://example.com/feed.xml", []byte(parseTestRSSFeed))
	if err != nil {
		t.Fatalf("parseFetchedEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	first := entries[0]
	if first.FeedID != 42 {
		t.Fatalf("FeedID = %d, want 42", first.FeedID)
	}
	if first.GUID != "post-1" {
		t.Fatalf("GUID = %q, want post-1", first.GUID)
	}
	if first.Content != "<p>Full content of post 1</p>" {
		t.Fatalf("Content = %q", first.Content)
	}
	if !strings.Contains(string(first.Categories), "Go") {
		t.Fatalf("Categories = %s, want Go category", first.Categories)
	}
	if !strings.Contains(string(first.Enclosures), "audio/mpeg") {
		t.Fatalf("Enclosures = %s, want audio/mpeg enclosure", first.Enclosures)
	}

	second := entries[1]
	if string(second.Categories) != "[]" {
		t.Fatalf("Categories = %q, want []", second.Categories)
	}
	if string(second.Enclosures) != "[]" {
		t.Fatalf("Enclosures = %q, want []", second.Enclosures)
	}
}

func TestParseFetchedEntriesAtom(t *testing.T) {
	entries, err := parseFetchedEntries(7, "https://example.com/atom.xml", []byte(parseTestAtomFeed))
	if err != nil {
		t.Fatalf("parseFetchedEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Author != "Alice" {
		t.Fatalf("Author = %q, want Alice", entry.Author)
	}
	if !strings.Contains(string(entry.Authors), "https://alice.example.com") {
		t.Fatalf("Authors = %s, want author URI", entry.Authors)
	}
	if !strings.Contains(string(entry.Contributors), "Charlie") {
		t.Fatalf("Contributors = %s, want Charlie", entry.Contributors)
	}
	if !strings.Contains(string(entry.Links), "alternate") {
		t.Fatalf("Links = %s, want alternate link", entry.Links)
	}
	if !strings.Contains(string(entry.Categories), "https://example.com/tags") {
		t.Fatalf("Categories = %s, want category scheme", entry.Categories)
	}
	if entry.Rights != "CC BY 4.0" {
		t.Fatalf("Rights = %q, want CC BY 4.0", entry.Rights)
	}
	if !strings.Contains(string(entry.Source), "Original Source") {
		t.Fatalf("Source = %s, want source title", entry.Source)
	}
}

func TestParseFetchedEntriesInvalidFeed(t *testing.T) {
	_, err := parseFetchedEntries(1, "https://example.com/feed.xml", []byte("not a feed"))
	if !errors.Is(err, ErrInvalidFeed) {
		t.Fatalf("expected ErrInvalidFeed, got %v", err)
	}
}
