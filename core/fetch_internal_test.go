package core

import (
	"testing"
	"time"
)

func TestFilterFeedsForFetchSkipsIntervalFeeds(t *testing.T) {
	now := testTime(2026, 4, 4, 12, 0, 0)
	recent := now.Add(-30 * testSecond)
	old := now.Add(-2 * testMinute)

	feeds := []*Feed{
		{ID: 1, Title: "recent", URL: "https://example.com/recent", FetchIntervalSec: 60, FetchedAt: &recent},
		{ID: 2, Title: "old", URL: "https://example.com/old", FetchIntervalSec: 60, FetchedAt: &old},
		{ID: 3, Title: "always", URL: "https://example.com/always"},
	}

	var events []FetchEvent
	notifier := newFetchNotifier(FetchObserverFunc(func(event FetchEvent) {
		events = append(events, event)
	}))

	filtered := filterFeedsForFetch(feeds, notifier, now)
	if len(filtered) != 2 {
		t.Fatalf("got %d feeds to fetch, want 2", len(filtered))
	}
	if filtered[0].ID != 2 || filtered[1].ID != 3 {
		t.Fatalf("unexpected feeds to fetch: %+v", filtered)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != FetchEventSkipped || events[0].SkipReason != FetchSkipInterval || events[0].FeedID != 1 {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

const (
	testSecond = time.Second
	testMinute = time.Minute
)

func testTime(year, month, day, hour, min, sec int) time.Time {
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
}
