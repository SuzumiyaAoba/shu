package app

import (
	"net/http"

	"github.com/SuzumiyaAoba/shu/core/coretest"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type fakeStore struct {
	coretest.BaseFakeStore
}

func newFakeStore() *fakeStore { return &fakeStore{} }

type trackingStore struct {
	*fakeStore
	closeFn func() error
}

func (s *trackingStore) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}
