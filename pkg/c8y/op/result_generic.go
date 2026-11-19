package op

import (
	"encoding/json"
	"errors"
	"iter"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
)

// ErrNoJSONDocument is returned when a result's data has no JSON document to
// decode (for example op.Result[int64]).
var ErrNoJSONDocument = errors.New("op: result data has no JSON document")

// errorResult copies the metadata of r into a failed result of another type.
func errorResult[T, R any](r Result[T]) Result[R] {
	return Result[R]{
		Status:     r.Status,
		HTTPStatus: r.HTTPStatus,
		Err:        r.Err,
		Retryable:  r.Retryable,
		Idempotent: r.Idempotent,
		Meta:       r.Meta,
		Attempts:   r.Attempts,
		Duration:   r.Duration,
		RequestID:  r.RequestID,
		Timestamp:  r.Timestamp,
	}
}

// Map converts the result's data with fn, keeping the status and HTTP
// metadata. fn is not called for a failed result; the error is kept.
//
//	names := result.Map(func(mo jsonmodels.ManagedObject) string { return mo.Name() })
func (r Result[T]) Map[R any](fn func(T) R) Result[R] {
	if r.IsError() {
		return errorResult[T, R](r)
	}
	out := errorResult[T, R](r)
	out.Data = fn(r.Data)
	return out
}

// FlatMap chains another operation on the result's data. fn is not called for
// a failed result; the error is kept.
func (r Result[T]) FlatMap[R any](fn func(T) Result[R]) Result[R] {
	if r.IsError() {
		return errorResult[T, R](r)
	}
	return fn(r.Data)
}

// Decode decodes the result's JSON document into U, for example a user-defined
// struct:
//
//	device, err := client.ManagedObjects.Get(ctx, id, opts).Decode[MyDevice]()
//
// It returns the result's error when the request failed, and ErrNoJSONDocument
// when the data has no JSON document.
func (r Result[T]) Decode[U any]() (U, error) {
	var out U
	if r.Err != nil {
		return out, r.Err
	}
	doc, ok := any(r.Data).(Unwrapper)
	if !ok {
		return out, ErrNoJSONDocument
	}
	err := json.Unmarshal(doc.Bytes(), &out)
	return out, err
}

// IterAs iterates the items of a collection result decoded into U. For a
// single-document result it yields that document. Decode errors are yielded
// per item; the result's own error is yielded first.
func (r Result[T]) IterAs[U any]() iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		var zero U
		if r.Err != nil {
			yield(zero, r.Err)
			return
		}
		if collection, ok := any(r.Data).(types.CollectionIterator); ok {
			for item := range collection.IterBytes() {
				var decoded U
				err := json.Unmarshal(item.Bytes(), &decoded)
				if !yield(decoded, err) {
					return
				}
			}
			return
		}
		if doc, ok := any(r.Data).(Unwrapper); ok {
			var decoded U
			err := json.Unmarshal(doc.Bytes(), &decoded)
			yield(decoded, err)
			return
		}
		yield(zero, ErrNoJSONDocument)
	}
}

// Seq iterates the items of a collection result, stopping at the first error.
// Use Items to see errors.
func (r Result[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item, err := range r.Items() {
			if err != nil || !yield(item) {
				return
			}
		}
	}
}

// Collect returns all items of a collection result, or the first error.
func (r Result[T]) Collect() ([]T, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return ToSlice(r.Items())
}
