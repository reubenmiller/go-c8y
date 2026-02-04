package pagination

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
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
type Iterator[T any] struct {
	items       iter.Seq[T]
	err         error
	totalCount  int64
	totalPages  int64
	previewDone bool
	previewFunc func() error // Closure to perform preview call
}

func (it *Iterator[T]) Items() iter.Seq[T] {
	return it.items
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

// Paginate creates an iterator that fetches pages and constructs items of type T
// The iterator is fully lazy - no API calls are made until Items() is called for iteration.
// Call Preview() to fetch metadata before iteration, or metadata will be populated from first page.
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
	iterator := &Iterator[T]{
		totalCount: -1,
		totalPages: -1,
	}

	// Create preview function closure
	iterator.previewFunc = func() error {
		if iterator.previewDone {
			return iterator.err
		}

		previewOpts := paginationOpts
		previewOpts.PageSize = 1
		previewOpts.CurrentPage = 1
		previewOpts.WithTotalPages = true
		previewOpts.WithTotalElements = true

		result := fetch(previewOpts)
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
	paginationOpts.PageSize = paginationOpts.OptimalPageSize()
	maxItems := paginationOpts.GetMaxItems()

	iterator.items = func(yield func(T) bool) {
		page := 1
		count := int64(0)
		for {
			// Copy and set current page for this iteration
			opts := paginationOpts
			opts.CurrentPage = page
			opts.WithTotalElements = true // Always request metadata
			opts.WithTotalPages = true

			result := fetch(opts)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}

			// Extract metadata on first page
			if !iterator.previewDone {
				iterator.previewDone = true
				if totalElements, ok := result.Meta["totalElements"].(int64); ok {
					iterator.totalCount = totalElements
				}
				if totalPages, ok := result.Meta["totalPages"].(int64); ok {
					iterator.totalPages = totalPages
				}
			}

			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := constructor(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			if next, ok := result.Meta["next"].(string); !ok || next == "" {
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}
