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
// fetch: function to fetch a page (receives page number, returns Result with collection)
// constructor: function to construct a T from JSON bytes
// maxItems: maximum number of items to return (0 for unlimited)
func Paginate[T any, D JSONDocument](
	ctx context.Context,
	fetch func(page int) op.Result[D],
	constructor func([]byte) T,
	maxItems int64,
) *Iterator[T] {
	iterator := &Iterator[T]{}

	iterator.items = func(yield func(T) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
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
