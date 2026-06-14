package pagination

import (
	"context"

	"github.com/destel/rill"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// pageEnvelope pairs a fetched page with the request that produced it, so the
// consumer can apply request-scoped filtering (Strategy.Accept) regardless of
// how far a prefetcher or parallel worker has run ahead.
type pageEnvelope[D any] struct {
	req    PageRequest
	result op.Result[D]
}

// countDocs returns the number of item documents in a successful page.
func countDocs[D JSONDocument](result op.Result[D]) int {
	if result.Err != nil {
		return 0
	}
	n := 0
	for range result.Data.Iter() {
		n++
	}
	return n
}

// withMeta sets the metadata flags every page request needs.
func withMeta(req PageRequest) PageRequest {
	req.WithTotalElements = true
	req.WithTotalPages = true
	return req
}

// walkPages drives the page walk and calls emit for each fetched page in order.
// emit returns false to stop early (consumer done, or an error it has surfaced).
// The first page is always fetched and emitted sequentially; the remainder is
// fanned out in parallel when the strategy's pages are independent and
// Concurrency > 1, otherwise walked sequentially with optional read-ahead.
//
// pageLimit (>0) caps the total number of pages fetched; it bounds over-fetching
// when MaxItems is set (the item-level cap in emit is the primary stop).
func walkPages[D JSONDocument](
	ctx context.Context,
	base PageRequest,
	strategy Strategy,
	fetch func(PageRequest) op.Result[D],
	readAhead int,
	pageLimit int,
	emit func(req PageRequest, result op.Result[D]) bool,
) {
	firstReq := withMeta(strategy.First(base))
	first := fetch(firstReq)
	if !emit(firstReq, first) {
		return
	}
	if first.Err != nil {
		return
	}
	if pageLimit > 0 && pageLimit <= 1 {
		return
	}

	next, more := strategy.Advance(firstReq, PageView{
		Meta:  first.Meta,
		Count: countDocs(first),
		Docs:  first.Data.Iter(),
	})
	if !more {
		return
	}

	// Parallel fan-out when the strategy declares independent pages.
	if ps, ok := strategy.(ParallelStrategy); ok && base.Concurrency > 1 {
		if plan, ok := ps.Plan(PageView{Meta: first.Meta}, base); ok {
			if pageLimit > 0 && len(plan) > pageLimit-1 {
				plan = plan[:pageLimit-1]
			}
			runParallel(ctx, plan, base.Concurrency, fetch, emit)
			return
		}
		// Plan declined (e.g. unknown total): fall back to sequential.
	}

	runSequential(strategy, fetch, readAhead, pageLimit, next, 1, emit)
}

// runParallel fetches the planned (independent) pages concurrently with bounded
// parallelism and emits them in request order. On an early stop it cancels
// pending fetches and drains the stream so no worker goroutine leaks.
func runParallel[D JSONDocument](
	ctx context.Context,
	plan []PageRequest,
	concurrency int,
	fetch func(PageRequest) op.Result[D],
	emit func(req PageRequest, result op.Result[D]) bool,
) {
	pctx, cancel := context.WithCancel(ctx)
	defer cancel()

	in := rill.FromSlice(plan, nil)
	out := rill.OrderedMap(in, concurrency, func(r PageRequest) (pageEnvelope[D], error) {
		if pctx.Err() != nil {
			return pageEnvelope[D]{}, pctx.Err()
		}
		return pageEnvelope[D]{req: r, result: fetch(withMeta(r))}, nil
	})

	stopped := false
	for env := range out {
		if stopped || env.Error != nil {
			continue // keep draining until the stream closes
		}
		if !emit(env.Value.req, env.Value.result) {
			stopped = true
			cancel() // skip not-yet-started fetches; finish draining
		}
	}
}

// runSequential walks pages one request at a time starting at startReq, which is
// the (pagesSoFar+1)-th page. With readAhead > 0 a producer goroutine fetches up
// to readAhead pages ahead while the consumer drains the current one; the
// producer never fetches past a page the strategy marks terminal, so a dry-run
// (empty first page) issues no speculative request.
func runSequential[D JSONDocument](
	strategy Strategy,
	fetch func(PageRequest) op.Result[D],
	readAhead int,
	pageLimit int,
	startReq PageRequest,
	pagesSoFar int,
	emit func(req PageRequest, result op.Result[D]) bool,
) {
	if readAhead <= 0 {
		req := startReq
		pages := pagesSoFar
		for {
			result := fetch(withMeta(req))
			if !emit(req, result) {
				return
			}
			if result.Err != nil {
				return
			}
			pages++
			if pageLimit > 0 && pages >= pageLimit {
				return
			}
			next, more := strategy.Advance(req, PageView{
				Meta:  result.Meta,
				Count: countDocs(result),
				Docs:  result.Data.Iter(),
			})
			if !more {
				return
			}
			req = next
		}
	}

	ch := make(chan pageEnvelope[D], readAhead)
	done := make(chan struct{})
	go func() {
		defer close(ch)
		req := startReq
		pages := pagesSoFar
		for {
			select {
			case <-done:
				return
			default:
			}
			result := fetch(withMeta(req))

			// Compute the continuation before handing the page off, so the
			// consumer has exclusive access to the result.
			var next PageRequest
			more := false
			if result.Err == nil {
				next, more = strategy.Advance(req, PageView{
					Meta:  result.Meta,
					Count: countDocs(result),
					Docs:  result.Data.Iter(),
				})
			}

			select {
			case ch <- pageEnvelope[D]{req: req, result: result}:
			case <-done:
				return
			}

			if result.Err != nil {
				return
			}
			pages++
			if pageLimit > 0 && pages >= pageLimit {
				return
			}
			if !more {
				return
			}
			req = next
		}
	}()

	for env := range ch {
		if !emit(env.req, env.result) {
			close(done)
			// Drain any buffered pages so the producer's pending send unblocks
			// and it can exit cleanly.
			go func() {
				for range ch {
				}
			}()
			return
		}
	}
}

// pageCap returns the number of pages needed to satisfy maxItems at the given
// page size (0 = unbounded).
func pageCap(maxItems int64, pageSize int) int {
	if maxItems <= 0 || pageSize <= 0 {
		return 0
	}
	return int((maxItems + int64(pageSize) - 1) / int64(pageSize))
}
