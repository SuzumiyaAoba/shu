package core

import (
	"context"
	"fmt"
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

	doc, err := html.Parse(strings.NewReader(string(body)))
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

// resolveURL resolves href relative to base. For simplicity, if href is already
// absolute it is returned as-is; otherwise it is joined with the base URL.
func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	// Strip path from base to get origin.
	if idx := strings.Index(base, "://"); idx != -1 {
		rest := base[idx+3:]
		if slash := strings.Index(rest, "/"); slash != -1 {
			if strings.HasPrefix(href, "/") {
				return base[:idx+3+slash] + href
			}
			// Relative path — join with directory of base.
			dir := rest[:strings.LastIndex(rest, "/")+1]
			return base[:idx+3] + dir + href
		}
	}
	if strings.HasPrefix(href, "/") {
		return strings.TrimRight(base, "/") + href
	}
	return base + "/" + href
}
