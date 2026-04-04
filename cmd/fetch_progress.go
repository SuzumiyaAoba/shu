package cmd

import (
	"io"
	"sync/atomic"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// fetchProgress implements core.FetchObserver and displays a single progress
// bar on the given writer while feeds are being fetched.
type fetchProgress struct {
	p       *mpb.Progress
	bar     *mpb.Bar
	total   int64
	fetched atomic.Int64
	skipped atomic.Int64
	errors  atomic.Int64
}

func newFetchProgress(w io.Writer, total int) *fetchProgress {
	p := mpb.New(mpb.WithOutput(w))
	fp := &fetchProgress{p: p, total: int64(total)}

	fp.bar = p.AddBar(int64(total),
		mpb.PrependDecorators(
			decor.Name("Fetching feeds "),
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), "done"),
		),
	)
	return fp
}

// OnFetchEvent is called by the service for every fetch lifecycle event.
func (fp *fetchProgress) OnFetchEvent(event core.FetchEvent) {
	switch event.Type {
	case core.FetchEventCompleted:
		fp.fetched.Add(1)
		if event.Err != nil {
			fp.errors.Add(1)
		}
		fp.bar.Increment()
	case core.FetchEventSkipped:
		fp.skipped.Add(1)
		fp.bar.Increment()
	}
}

// Wait blocks until the progress bar finishes rendering.
func (fp *fetchProgress) Wait() {
	fp.p.Wait()
}
