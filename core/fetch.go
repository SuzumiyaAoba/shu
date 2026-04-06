package core

import (
	"context"

	"github.com/SuzumiyaAoba/shu/core/fetch"
	"github.com/SuzumiyaAoba/shu/model"
)

// FetchFeed downloads and parses the RSS/Atom feed identified by feedID.
func (s *Service) FetchFeed(ctx context.Context, feedID int64) ([]*model.Entry, error) {
	return s.fetcher.FetchFeed(ctx, feedID)
}

func (s *Service) FetchFeedWithObserver(ctx context.Context, feedID int64, observer fetch.Observer) ([]*model.Entry, error) {
	return s.fetcher.FetchFeedWithObserver(ctx, feedID, observer)
}

func (s *Service) FetchAll(ctx context.Context) (int, error) {
	return s.fetcher.FetchAll(ctx)
}

func (s *Service) FetchAllWithObserver(ctx context.Context, observer fetch.Observer) (int, error) {
	return s.fetcher.FetchAllWithObserver(ctx, observer)
}
