package core

import "testing"

func TestValidateFeedURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid
		{"https", "https://example.com/feed.xml", false},
		{"http", "http://example.com/rss", false},
		{"with port", "https://example.com:8080/feed", false},
		{"with path and query", "https://example.com/feed?format=rss", false},

		// Invalid scheme
		{"ftp", "ftp://example.com/feed.xml", true},
		{"file", "file:///etc/passwd", true},
		{"javascript", "javascript:alert(1)", true},
		{"empty scheme", "://example.com", true},
		{"no scheme", "example.com/feed.xml", true},

		// Missing host
		{"empty", "", true},
		{"scheme only", "https://", true},

		// Private/loopback addresses
		{"localhost", "https://localhost/feed.xml", true},
		{"127.0.0.1", "http://127.0.0.1:8080/feed", true},
		{"10.x.x.x", "http://10.0.0.1/feed", true},
		{"172.16.x.x", "http://172.16.0.1/feed", true},
		{"192.168.x.x", "http://192.168.1.1/feed", true},
		{"::1", "http://[::1]/feed", true},
		{"link-local", "http://169.254.1.1/feed", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeedURL(tt.url, false)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateFeedURL(%q, false) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateFeedURL(%q, false) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestValidateFeedURLAllowPrivate(t *testing.T) {
	privateURLs := []string{
		"http://localhost/feed",
		"http://127.0.0.1:8080/feed",
		"http://192.168.1.1/feed",
	}
	for _, u := range privateURLs {
		if err := ValidateFeedURL(u, true); err != nil {
			t.Errorf("ValidateFeedURL(%q, true) = %v, want nil", u, err)
		}
	}

	// Non-HTTP schemes are still rejected even with allowPrivate.
	if err := ValidateFeedURL("ftp://localhost/feed", true); err == nil {
		t.Error("ValidateFeedURL(ftp, true) = nil, want error")
	}
}
