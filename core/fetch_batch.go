package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const fetchWorkerCount = 10

// FetchAll fetches every registered feed concurrently (up to 10 at a time) and
// returns the total number of new entries stored across all feeds.
//
// If an individual feed fails to fetch (network error, parse error, etc.), the
// error is logged and the method continues with the remaining feeds. This
// ensures that a single broken feed does not block updates for others.
//
// Feeds that have a per-feed interval set and were fetched more recently than
// that interval are skipped.
func (f *Fetcher) FetchAll(ctx context.Context) (int, error) {
	return f.FetchAllWithObserver(ctx, nil)
}

// FetchAllWithObserver fetches all eligible feeds while emitting structured
// progress events to observer.
func (f *Fetcher) FetchAllWithObserver(ctx context.Context, observer FetchObserver) (int, error) {
	feeds, err := f.store.ListFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("list feeds: %w", err)
	}

	notifier := newFetchNotifier(observer)
	toFetch := filterFeedsForFetch(feeds, notifier, time.Now())
	if len(toFetch) == 0 {
		return 0, nil
	}

	return f.fetchFeedsConcurrently(ctx, toFetch, notifier)
}

func filterFeedsForFetch(feeds []*Feed, notifier *fetchNotifier, now time.Time) []*Feed {
	toFetch := make([]*Feed, 0, len(feeds))
	for _, feed := range feeds {
		if shouldSkipFeedInterval(feed, now) {
			notifier.skipped(feed, FetchSkipInterval)
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

func (f *Fetcher) fetchFeedsConcurrently(ctx context.Context, feeds []*Feed, notifier *fetchNotifier) (int, error) {
	jobs := make(chan *Feed)
	workers := min(fetchWorkerCount, len(feeds))
	var (
		total     atomic.Int64
		wg        sync.WaitGroup
		errMu     sync.Mutex
		fetchErrs = make([]error, 0, min(len(feeds), 64))
	)

	worker := func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				errMu.Lock()
				fetchErrs = append(fetchErrs, fmt.Errorf("worker panic: %v", r))
				errMu.Unlock()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case feed, ok := <-jobs:
				if !ok {
					return
				}

				entries, err := f.fetchFeed(ctx, feed, notifier)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					f.logger.With("feed_id", feed.ID, "feed_url", feed.URL).Error("failed to fetch feed", "error", err)
					errMu.Lock()
					fetchErrs = append(fetchErrs, err)
					errMu.Unlock()
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

	return int(total.Load()), errors.Join(fetchErrs...)
}

func (s *Service) FetchAll(ctx context.Context) (int, error) {
	return s.fetcher.FetchAll(ctx)
}

func (s *Service) FetchAllWithObserver(ctx context.Context, observer FetchObserver) (int, error) {
	return s.fetcher.FetchAllWithObserver(ctx, observer)
}
