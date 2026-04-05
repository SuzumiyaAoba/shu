package core

import (
	"fmt"
	"net"
	"net/url"
)

// ValidateFeedURL checks that rawURL is a valid HTTP(S) URL that does not
// point to a private or loopback address. It is intended to be called at the
// system boundary (AddFeed, DiscoverFeeds) to reject clearly invalid or
// potentially dangerous URLs early.
//
// When allowPrivate is true, the private/loopback address check is skipped.
// This is useful for development and testing with local servers.
func ValidateFeedURL(rawURL string, allowPrivate bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q: only http and https are allowed", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	if !allowPrivate {
		host := u.Hostname()
		if isPrivateHost(host) {
			return fmt.Errorf("URL points to a private/loopback address: %s", host)
		}
	}
	return nil
}

// isPrivateHost returns true if host is a loopback, link-local, or
// RFC 1918 private address. It also recognises the "localhost" hostname.
func isPrivateHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
