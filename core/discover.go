package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FeedDiscovery owns feed URL discovery from HTML pages.
type FeedDiscovery struct {
	client                *http.Client
	allowPrivateAddresses bool
}

// NewFeedDiscovery creates a feed discovery service.
func NewFeedDiscovery(client *http.Client) *FeedDiscovery {
	return &FeedDiscovery{client: normalizeHTTPClient(client)}
}

func (d *FeedDiscovery) setHTTPClient(client *http.Client) {
	d.client = normalizeHTTPClient(client)
}

// DiscoverFeeds fetches the HTML page at the given URL and extracts feed URLs
// from <link rel="alternate"> elements with RSS/Atom MIME types.
func (d *FeedDiscovery) DiscoverFeeds(ctx context.Context, pageURL string) ([]string, error) {
	if err := ValidateFeedURL(pageURL, d.allowPrivateAddresses); err != nil {
		return nil, fmt.Errorf("validate page URL: %w", err)
	}

	body, _, err := fetchBodyConditional(ctx, d.client, pageURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("fetch page %s: %w", pageURL, err)
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var feeds []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			rel, typ, href := "", "", ""
			for _, a := range n.Attr {
				switch a.Key {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "type":
					typ = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				}
			}
			if rel == "alternate" && href != "" && isFeedType(typ) {
				feeds = append(feeds, resolveURL(pageURL, href))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return feeds, nil
}

func isFeedType(typ string) bool {
	switch typ {
	case "application/rss+xml", "application/atom+xml", "application/feed+json", "application/xml", "text/xml":
		return true
	}
	return false
}

// resolveURL resolves href relative to base.
func resolveURL(base, href string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return baseURL.ResolveReference(ref).String()
}

func (s *Service) DiscoverFeeds(ctx context.Context, pageURL string) ([]string, error) {
	return s.discovery.DiscoverFeeds(ctx, pageURL)
}
