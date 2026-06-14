package pagination

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// JSONDocument represents any type that can provide iteration over JSON documents
// This is satisfied by all jsonmodels types (Alarm, Operation, Event, etc.)
type JSONDocument interface {
	Iter() iter.Seq[jsondoc.JSONDoc]
}

// Iterator provides iteration over paginated results of type T
// The iterator is fully lazy - no API calls are made until Items() is called.
// Call Preview() to fetch metadata (totalCount, totalPages) before iteration,
// which allows inspection and confirmation workflows.
//
// Error handling: Items() returns iter.Seq2[T, error]. Always check the error
// value in the loop — errors mid-iteration (e.g. a failed page fetch) will be
// yielded as the second value and must be handled explicitly:
//
//	for item, err := range iter.Items() {
//		if err != nil {
//			// handle or break
//		}
//		// use item
//	}
//
// Use Seq() only when integrating with libraries that require iter.Seq[T] and
// you are willing to silently discard mid-iteration errors.
type Iterator[T any] struct {
	items       iter.Seq2[T, error]
	pages       iter.Seq2[jsondoc.JSONDoc, error]
	err         error
	totalCount  int64
	totalPages  int64
	previewDone bool
	previewFunc func() error // Closure to perform preview call
}

// Items returns an iterator that yields each item together with any error
// encountered while fetching that page. Always check the error value:
//
//	for item, err := range it.Items() {
//		if err != nil { ... }
//	}
func (it *Iterator[T]) Items() iter.Seq2[T, error] {
	return it.items
}

// Seq returns an iterator that yields only successful items, discarding errors.
// This is provided for compatibility with libraries that expect iter.Seq[T].
// Use Items() if you need to handle errors from the iteration.
func (it *Iterator[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item, err := range it.items {
			if err != nil {
				// Skip errors - they're lost in this conversion
				continue
			}
			if !yield(item) {
				return
			}
		}
	}
}

// Pages returns an iterator that yields each page's full response envelope
// (the un-plucked collection wrapper: items array, statistics and paging
// links) exactly as received from the server. Use it for raw output where the
// whole document — not the extracted items — is wanted. Pagination follows the
// same rules as Items() (page size, MaxItems, total-page limits), so a plain
// query yields one page and an unbounded one yields every page. Like Items(),
// always check the error value:
//
//	for doc, err := range it.Pages() {
//		if err != nil { ... }
//	}
//
// Pages and Items are independent views over the same query — range over one
// or the other, not both. Returns an empty sequence for iterators created via
// NewIterator (which have no underlying page envelopes).
func (it *Iterator[T]) Pages() iter.Seq2[jsondoc.JSONDoc, error] {
	if it.pages == nil {
		return func(yield func(jsondoc.JSONDoc, error) bool) {}
	}
	return it.pages
}

func (it *Iterator[T]) Err() error {
	return it.err
}

// TotalCount returns the total number of items available
// Returns -1 until Preview() or first iteration populates this value
func (it *Iterator[T]) TotalCount() int64 {
	return it.totalCount
}

// TotalPages returns the total number of pages available
// Returns -1 until Preview() or first iteration populates this value
func (it *Iterator[T]) TotalPages() int64 {
	return it.totalPages
}

// Preview performs a lightweight API call (pageSize=1, withTotalElements=true)
// to fetch metadata about the collection without retrieving all items.
// This allows inspection of TotalCount() and TotalPages() before committing to full iteration.
// Returns any error encountered. Safe to call multiple times (only executes once).
func (it *Iterator[T]) Preview() error {
	if it.previewDone {
		return it.err
	}
	if it.previewFunc != nil {
		return it.previewFunc()
	}
	return nil
}

// NewIterator wraps a pre-built iter.Seq2 in an Iterator.
// TotalCount and TotalPages will return -1 (not applicable for non-paginated sources).
func NewIterator[T any](items iter.Seq2[T, error]) *Iterator[T] {
	return &Iterator[T]{
		items:      items,
		totalCount: -1,
		totalPages: -1,
	}
}

// NewErrorIterator returns an iterator that yields a single error and nothing
// else. Used to surface up-front failures (e.g. an inapplicable pagination
// strategy) through the normal Items()/Pages()/Err() interface.
func NewErrorIterator[T any](err error) *Iterator[T] {
	return &Iterator[T]{
		items:      func(yield func(T, error) bool) { yield(*new(T), err) },
		pages:      func(yield func(jsondoc.JSONDoc, error) bool) { yield(jsondoc.Empty(), err) },
		err:        err,
		totalCount: -1,
		totalPages: -1,
	}
}

// Paginate creates an iterator using classic offset pagination. It is a thin
// wrapper over PaginateWith with OffsetStrategy, kept for the many callers that
// don't need a keyset strategy.
//
// paginationOpts: pagination options (passed by value - will not modify caller's copy)
// fetch: function to fetch a page (returns Result with collection)
// constructor: function to construct a T from JSON bytes
func Paginate[T any, D JSONDocument](
	ctx context.Context,
	paginationOpts PaginationOptions,
	fetch func(opts PaginationOptions) op.Result[D],
	constructor func([]byte) T,
) *Iterator[T] {
	return PaginateWith(
		ctx,
		PageRequest{PaginationOptions: paginationOpts},
		OffsetStrategy{},
		func(req PageRequest) op.Result[D] { return fetch(req.PaginationOptions) },
		constructor,
	)
}

// PaginateWith creates an iterator driven by the given Strategy. The iterator is
// fully lazy — no API calls happen until Items() or Pages() is ranged. The
// strategy decides the first request, how to advance (or stop), and which items
// to emit (boundary dedup); the fetch closure applies the request — including
// any keyset cursor in PageRequest — to the entity's own options.
//
// base: the starting request (pagination options + any cursor seed)
// strategy: drives the page walk (OffsetStrategy reproduces classic paging)
// fetch: function to fetch one page for a PageRequest
// constructor: function to construct a T from JSON bytes
func PaginateWith[T any, D JSONDocument](
	ctx context.Context,
	base PageRequest,
	strategy Strategy,
	fetch func(req PageRequest) op.Result[D],
	constructor func([]byte) T,
) *Iterator[T] {
	iterator := &Iterator[T]{
		totalCount: -1,
		totalPages: -1,
	}

	captureMeta := func(result op.Result[D]) {
		if iterator.previewDone {
			return
		}
		iterator.previewDone = true
		if totalElements, ok := result.Meta["totalElements"].(int64); ok {
			iterator.totalCount = totalElements
		}
		if totalPages, ok := result.Meta["totalPages"].(int64); ok {
			iterator.totalPages = totalPages
		}
	}

	// Create preview function closure
	iterator.previewFunc = func() error {
		if iterator.previewDone {
			return iterator.err
		}

		previewReq := base
		previewReq.PageSize = 1
		previewReq.CurrentPage = 1
		previewReq.WithTotalPages = true
		previewReq.WithTotalElements = true

		result := fetch(previewReq)
		iterator.previewDone = true

		if result.Err != nil {
			iterator.err = result.Err
			return iterator.err
		}

		if totalElements, ok := result.Meta["totalElements"].(int64); ok {
			iterator.totalCount = totalElements
		}
		if totalPages, ok := result.Meta["totalPages"].(int64); ok {
			iterator.totalPages = totalPages
		}

		return nil
	}

	// Set optimal page size once
	base.PageSize = base.OptimalPageSize()
	maxItems := base.GetMaxItems()
	readAhead := resolveReadAhead(base.PaginationOptions, strategy)
	pageLimit := pageCap(maxItems, base.PageSize)

	iterator.items = func(yield func(T, error) bool) {
		count := int64(0)
		walkPages(ctx, base, strategy, fetch, readAhead, pageLimit,
			func(req PageRequest, result op.Result[D]) bool {
				if result.Err != nil {
					iterator.err = result.Err
					yield(*new(T), result.Err) // surface the error to the consumer
					return false
				}
				captureMeta(result)
				for doc := range result.Data.Iter() {
					if !strategy.Accept(req, doc) {
						continue // boundary duplicate dropped by the strategy
					}
					if maxItems > 0 && count >= maxItems {
						return false
					}
					if !yield(constructor(doc.Bytes()), nil) {
						return false
					}
					count++
				}
				return true
			})
	}

	// pages mirrors items but yields each page's full response envelope instead
	// of the extracted items. MaxItems is honoured at page granularity (stop
	// after the page that reaches the cap). Raw pages are always walked
	// sequentially (no boundary dedup is applied to whole envelopes).
	iterator.pages = func(yield func(jsondoc.JSONDoc, error) bool) {
		count := int64(0)
		walkPages(ctx, base, strategy, fetch, readAhead, pageLimit,
			func(req PageRequest, result op.Result[D]) bool {
				if result.Err != nil {
					iterator.err = result.Err
					yield(jsondoc.Empty(), result.Err)
					return false
				}
				captureMeta(result)

				// Emit the whole page envelope as received. A zero-result query
				// still has a body ({"managedObjects":[],...}) and is emitted;
				// only a truly empty body (e.g. the 204 a dry-run short-circuit
				// returns) is skipped so it produces no stray document.
				if len(result.Response) > 0 {
					if !yield(jsondoc.New(result.Response), nil) {
						return false
					}
				}

				pageItems := countDocs(result)
				count += int64(pageItems)
				if pageItems == 0 {
					slog.Debug("Stopping pagination as results array is empty")
					return false
				}
				if maxItems > 0 && count >= maxItems {
					return false
				}
				return true
			})
	}

	return iterator
}
