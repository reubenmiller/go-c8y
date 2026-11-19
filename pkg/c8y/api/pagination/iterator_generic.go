package pagination

import (
	"encoding/json"
	"errors"
	"iter"
)

// ErrNotJSON is returned by Iterator.As for items without a JSON document.
var ErrNotJSON = errors.New("pagination: item has no JSON document")

// Map iterates all pages, converting each item with fn. It stops at the first
// page error, which it yields.
//
//	for name, err := range client.ManagedObjects.ListAll(ctx, opts).Map(nameOf) { ... }
func (it *Iterator[T]) Map[R any](fn func(T) R) iter.Seq2[R, error] {
	return func(yield func(R, error) bool) {
		for item, err := range it.Items() {
			var zero R
			if err != nil {
				yield(zero, err)
				return
			}
			if !yield(fn(item), nil) {
				return
			}
		}
	}
}

// As iterates all pages, decoding each item into U (for example a
// user-defined struct). It stops at the first page error, which it yields.
func (it *Iterator[T]) As[U any]() iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		for item, err := range it.Items() {
			var out U
			if err != nil {
				yield(out, err)
				return
			}
			doc, ok := any(item).(interface{ Bytes() []byte })
			if !ok {
				if !yield(out, ErrNotJSON) {
					return
				}
				continue
			}
			if !yield(out, json.Unmarshal(doc.Bytes(), &out)) {
				return
			}
		}
	}
}
