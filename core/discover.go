package core

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// DiscoverFeeds fetches the HTML page at the given URL and extracts feed URLs
// from <link rel="alternate"> elements with RSS/Atom MIME types.
func (s *Service) DiscoverFeeds(ctx context.Context, pageURL string) ([]string, error) {
	body, err := s.fetchBody(ctx, pageURL)
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
