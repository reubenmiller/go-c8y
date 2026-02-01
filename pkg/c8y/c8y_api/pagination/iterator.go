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
type Iterator[T any] struct {
	items iter.Seq[T]
	err   error
}

func (it *Iterator[T]) Items() iter.Seq[T] {
	return it.items
}

func (it *Iterator[T]) Err() error {
	return it.err
}

// Paginate creates an iterator that fetches pages and constructs items of type T
// paginationOpts: pagination options (passed by value - will not modify caller's copy)
// fetch: function to fetch a page (returns Result with collection)
// constructor: function to construct a T from JSON bytes
func Paginate[T any, D JSONDocument](
	ctx context.Context,
	paginationOpts PaginationOptions,
	fetch func() op.Result[D],
	constructor func([]byte) T,
) *Iterator[T] {
	iterator := &Iterator[T]{}

	// Set optimal page size if not already set
	paginationOpts.PageSize = paginationOpts.OptimalPageSize()
	maxItems := paginationOpts.GetMaxItems()

	iterator.items = func(yield func(T) bool) {
		page := 1
		count := int64(0)
		for {
			paginationOpts.CurrentPage = page
			result := fetch()
			if result.Err != nil {
				iterator.err = result.Err
				return
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

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}
