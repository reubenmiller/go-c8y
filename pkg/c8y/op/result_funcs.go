package op

import (
	"context"
	"net/http"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
)

// The functions below build a Result. They are package functions rather than
// methods on purpose: see the package documentation.

// WithHTTPStatus returns r with the HTTP status code set.
func WithHTTPStatus[T any](r Result[T], status int) Result[T] {
	r.HTTPStatus = status
	return r
}

// WithRequestID returns r with the request correlation ID set.
func WithRequestID[T any](r Result[T], id string) Result[T] {
	r.RequestID = id
	return r
}

// WithAttempts returns r with the number of retry attempts set.
func WithAttempts[T any](r Result[T], attempts int) Result[T] {
	r.Attempts = attempts
	return r
}

// WithDuration returns r with the operation duration set.
func WithDuration[T any](r Result[T], duration time.Duration) Result[T] {
	r.Duration = duration
	return r
}

// WithRequest returns r with the HTTP request set.
func WithRequest[T any](r Result[T], req *http.Request) Result[T] {
	r.Request = req
	return r
}

// WithResponse returns r with the raw response body (the full, un-plucked
// envelope) attached.
func WithResponse[T any](r Result[T], body []byte) Result[T] {
	r.Response = body
	return r
}

// WithExecutor returns r with the function that runs a deferred operation.
func WithExecutor[T any](r Result[T], executor func(context.Context) Result[T]) Result[T] {
	r.executor = executor
	return r
}

// WithMeta returns r with the metadata key set.
func WithMeta[T any](r Result[T], key string, value any) Result[T] {
	if r.Meta == nil {
		r.Meta = make(Meta)
	}
	r.Meta[key] = value
	return r
}

// ExecuteOrDefer runs r immediately unless deferred execution is enabled in
// the context, in which case r is returned with its executor for later:
//
//	return op.ExecuteOrDefer(ctx, op.WithExecutor(op.Result[T]{}, func(ctx context.Context) op.Result[T] {
//	    // implementation
//	}))
func ExecuteOrDefer[T any](ctx context.Context, r Result[T]) Result[T] {
	if ctxhelpers.IsDeferredExecution(ctx) {
		return r
	}
	return r.Execute(ctx)
}

// Meta is the metadata of a Result. Its accessors read the collection
// statistics the server reported.
type Meta map[string]any

// TotalElements returns the total number of matching elements reported by the
// server for a collection result. It is only populated when the request
// included WithTotalElements: true; otherwise it is 0.
func (m Meta) TotalElements() int64 {
	v, _ := m["totalElements"].(int64)
	return v
}

// TotalPages returns the total number of pages for the current page size. It
// is only populated when the request included WithTotalPages: true;
// otherwise it is 0.
func (m Meta) TotalPages() int64 {
	v, _ := m["totalPages"].(int64)
	return v
}

// CurrentPage returns the page number of the result within the collection,
// or 0 if the result is not a collection.
func (m Meta) CurrentPage() int64 {
	v, _ := m["currentPage"].(int64)
	return v
}

// PageSize returns the maximum number of items per page, or 0 if the result
// is not a collection.
func (m Meta) PageSize() int64 {
	v, _ := m["pageSize"].(int64)
	return v
}

// HasNextPage reports whether the server indicated a next page is available.
func (m Meta) HasNextPage() bool {
	next, _ := m["next"].(string)
	return next != ""
}
