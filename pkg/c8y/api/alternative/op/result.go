package op

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
)

// Status represents the outcome of an operation
type Status string

const (
	StatusOK        Status = "OK"        // Existing resource retrieved
	StatusCreated   Status = "Created"   // New resource created
	StatusUpdated   Status = "Updated"   // Existing resource modified
	StatusNoContent Status = "NoContent" // Response does not contain any content
	StatusSkipped   Status = "Skipped"   // Operation skipped (e.g., no-op update)
	StatusDuplicate Status = "Duplicate" // Resource already exists (conflict)
	StatusNoMatch   Status = "NoMatch"   // No matches found
	StatusFailed    Status = "Failed"    // Operation failed
)

// Result wraps operation results with comprehensive metadata
type Result[T any] struct {
	Data       T      // The actual result data
	Status     Status // Operation outcome
	HTTPStatus int    // HTTP status code
	Err        error  // Error if operation failed

	// Operation characteristics
	Retryable  bool           // Whether the operation can be retried
	Idempotent bool           // Whether operation is idempotent
	Meta       map[string]any // Additional metadata

	// Operation tracking
	Attempts  int           // Number of retry attempts
	Duration  time.Duration // Total operation duration
	RequestID string        // Correlation ID for debugging
	Timestamp time.Time     // When the operation completed

	// Request details for inspection (e.g., dry run, debugging)
	Request *http.Request // The HTTP request that was (or would be) sent

	// Deferred execution support
	executor func(context.Context) Result[T] // Function to execute the actual operation
}

// OK creates a successful result for retrieved resource
func OK[T any](data T, meta ...map[string]any) Result[T] {
	r := Result[T]{
		Data:      data,
		Status:    StatusOK,
		Timestamp: time.Now(),
		Meta:      make(map[string]any),
	}
	if len(meta) > 0 && meta[0] != nil {
		r.Meta = meta[0]
	}
	return r
}

// Created creates a result for newly created resource
func Created[T any](data T, meta ...map[string]any) Result[T] {
	r := Result[T]{
		Data:       data,
		Status:     StatusCreated,
		HTTPStatus: 201,
		Idempotent: false,
		Timestamp:  time.Now(),
		Meta:       make(map[string]any),
	}
	if len(meta) > 0 && meta[0] != nil {
		r.Meta = meta[0]
	}
	return r
}

// Updated creates a result for updated resource
func Updated[T any](data T, meta ...map[string]any) Result[T] {
	r := Result[T]{
		Data:       data,
		Status:     StatusUpdated,
		HTTPStatus: 200,
		Timestamp:  time.Now(),
		Meta:       make(map[string]any),
	}
	if len(meta) > 0 && meta[0] != nil {
		r.Meta = meta[0]
	}
	return r
}

func NoContent[T any](data T, meta ...map[string]any) Result[T] {
	r := Result[T]{
		Data:       data,
		Status:     StatusNoContent,
		HTTPStatus: http.StatusNoContent,
		Idempotent: false,
		Timestamp:  time.Now(),
		Meta:       make(map[string]any),
	}
	if len(meta) > 0 && meta[0] != nil {
		r.Meta = meta[0]
	}
	return r
}

// Skipped creates a result for skipped operation
func Skipped[T any](data T, reason string) Result[T] {
	return Result[T]{
		Data:       data,
		Status:     StatusSkipped,
		Idempotent: true,
		Timestamp:  time.Now(),
		Meta: map[string]any{
			"reason": reason,
		},
	}
}

// Duplicate creates a result for duplicate/conflict scenarios
func Duplicate[T any](data T, meta ...map[string]any) Result[T] {
	r := Result[T]{
		Data:       data,
		Status:     StatusDuplicate,
		HTTPStatus: 409,
		Idempotent: true,
		Timestamp:  time.Now(),
		Meta:       make(map[string]any),
	}
	if len(meta) > 0 && meta[0] != nil {
		r.Meta = meta[0]
	}
	return r
}

// Not found, though generally this is not an error
func NoMatch[T any](meta ...map[string]any) Result[T] {
	var zero T
	r := Result[T]{
		Data:       zero,
		Status:     StatusNoMatch,
		Idempotent: true,
		Timestamp:  time.Now(),
		Meta:       make(map[string]any),
	}
	return r
}

// Failed creates a failed result with error
func Failed[T any](err error, retryable bool) Result[T] {
	var zero T
	return Result[T]{
		Data:      zero,
		Status:    StatusFailed,
		Err:       err,
		Retryable: retryable,
		Timestamp: time.Now(),
		Meta:      make(map[string]any),
	}
}

// WithHTTPStatus sets the HTTP status code
func (r Result[T]) WithHTTPStatus(status int) Result[T] {
	r.HTTPStatus = status
	return r
}

// WithRequestID sets the request correlation ID
func (r Result[T]) WithRequestID(id string) Result[T] {
	r.RequestID = id
	return r
}

// WithAttempts sets the number of retry attempts
func (r Result[T]) WithAttempts(attempts int) Result[T] {
	r.Attempts = attempts
	return r
}

// WithDuration sets the operation duration
func (r Result[T]) WithDuration(duration time.Duration) Result[T] {
	r.Duration = duration
	return r
}

// WithRequest sets the HTTP request
func (r Result[T]) WithRequest(req *http.Request) Result[T] {
	r.Request = req
	return r
}

// WithExecutor stores the execution function for deferred execution
func (r Result[T]) WithExecutor(executor func(context.Context) Result[T]) Result[T] {
	r.executor = executor
	return r
}

// IsDeferred returns true if this result has deferred execution
func (r Result[T]) IsDeferred() bool {
	return r.executor != nil
}

// Execute runs the deferred operation if present, otherwise returns self
// This allows: prepared := client.Delete(ctx, id); result := prepared.Execute(ctx)
func (r Result[T]) Execute(ctx context.Context) Result[T] {
	if r.executor != nil {
		return r.executor(ctx)
	}
	return r // Already executed
}

// ExecuteOrDefer executes the result immediately if deferred execution is not enabled in the context,
// otherwise returns the result with its executor for later execution.
// This is the standard pattern for operations that support deferred execution:
//
//	return op.Result[T]{}.WithExecutor(func(ctx context.Context) op.Result[T] {
//	    // implementation
//	}).WithMeta(...).ExecuteOrDefer(ctx)
func (r Result[T]) ExecuteOrDefer(ctx context.Context) Result[T] {
	if ctxhelpers.IsDeferredExecution(ctx) {
		return r
	}
	return r.Execute(ctx)
}

// WithMeta adds metadata to the result
func (r Result[T]) WithMeta(key string, value any) Result[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[key] = value
	return r
}

// IgnoreNotFound clears any error if the HTTP status is 404 Not Found.
// This is useful for DELETE operations where a 404 indicates the resource
// is already absent (desired state achieved).
//
// Example:
//
//	deleteResult := client.Delete(ctx, id).IgnoreNotFound()
//	if deleteResult.Err != nil {
//	    // Only real errors, not 404s
//	}
func (r Result[T]) IgnoreNotFound() Result[T] {
	if r.HTTPStatus == 404 {
		r.Err = nil
		r = r.WithMeta("ignoredStatus", 404)
	}
	return r
}

// IgnoreStatusCodes clears any error if the HTTP status matches any of the provided codes.
// This is useful for operations where certain HTTP status codes are acceptable.
//
// Example:
//
//	result := client.Update(ctx, data).IgnoreStatusCodes(404, 409)
//	if result.Err != nil {
//	    // Only errors not related to 404 or 409
//	}
func (r Result[T]) IgnoreStatusCodes(codes ...int) Result[T] {
	for _, code := range codes {
		if r.HTTPStatus == code {
			r.Err = nil
			r = r.WithMeta("ignoredStatus", code)
			break
		}
	}
	return r
}

// IsError returns true if the result contains an error
func (r Result[T]) IsError() bool {
	return r.Err != nil || r.Status == StatusFailed
}

// IsSuccess returns true if the operation succeeded
func (r Result[T]) IsSuccess() bool {
	return r.Status == StatusOK || r.Status == StatusCreated || r.Status == StatusUpdated || r.Status == StatusSkipped
}

// IsNotFound returns true if the HTTP status is 404
func (r Result[T]) IsNotFound() bool {
	return r.HTTPStatus == 404
}

// IsRetryable returns true if the HTTP status indicates a retryable error
// Retryable errors include server errors (5xx) and rate limiting (429)
// Note: For 429 responses, clients should respect the Retry-After header to determine
// when to retry. This method only indicates that the error category is retryable.
func (r Result[T]) IsRetryable() bool {
	return r.HTTPStatus >= 500 || r.HTTPStatus == 429
}

// Unwrap returns data and error (compatible with standard error handling)
func (r Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}

// MapResult transforms the data using a mapping function
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.IsError() {
		return Result[U]{
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

	return Result[U]{
		Data:       fn(r.Data),
		Status:     r.Status,
		HTTPStatus: r.HTTPStatus,
		Retryable:  r.Retryable,
		Idempotent: r.Idempotent,
		Meta:       r.Meta,
		Attempts:   r.Attempts,
		Duration:   r.Duration,
		RequestID:  r.RequestID,
		Timestamp:  r.Timestamp,
	}
}

// FlatMap transforms and flattens the result
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.IsError() {
		return Result[U]{
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

	return fn(r.Data)
}

// Combine merges two results, preferring errors
func Combine[T, U, V any](r1 Result[T], r2 Result[U], fn func(T, U) V) Result[V] {
	if r1.IsError() {
		return Result[V]{
			Status:    r1.Status,
			Err:       r1.Err,
			Timestamp: time.Now(),
		}
	}
	if r2.IsError() {
		return Result[V]{
			Status:    r2.Status,
			Err:       r2.Err,
			Timestamp: time.Now(),
		}
	}

	return Result[V]{
		Data:      fn(r1.Data, r2.Data),
		Status:    StatusOK,
		Timestamp: time.Now(),
	}
}

// First returns a Result containing the first item from an iterator, or NoMatch if empty.
// If the iterator yields an error, First returns a Failed result with that error.
func First[T any](items iter.Seq2[T, error]) (Result[T], bool) {
	for item, err := range items {
		if err != nil {
			return Failed[T](err, false), true
		}
		return OK(item), true
	}
	return NoMatch[T](), false
}

// Unwrapper is a constraint for types that can provide their raw bytes
type Unwrapper interface {
	Bytes() []byte
}

// IterAs transforms an iterator of JSONDoc-based items into an iterator of type U by unmarshaling.
// Items that fail to unmarshal are skipped.
// This is a convenience method for Result types containing collection data.
//
// Example:
//
//	type CustomMeasurement struct {
//	    ID string `json:"id"`
//	    Temperature struct {
//	        Value float64 `json:"value"`
//	    } `json:"c8y_Temperature"`
//	}
//	collection := client.Measurements.List(ctx, opts)
//	for m := range collection.IterAs[CustomMeasurement]() {
//	    fmt.Printf("Temp: %.2f\n", m.Temperature.Value)
//	}
func IterAs[U any, T Unwrapper](r Result[T]) iter.Seq[U] {
	return func(yield func(U) bool) {
		// Type switch to check if Data has an Iter() method
		type Iterable interface {
			Iter() iter.Seq[T]
		}

		if iterable, ok := any(r.Data).(Iterable); ok {
			for item := range iterable.Iter() {
				var decoded U
				if err := json.Unmarshal(item.Bytes(), &decoded); err != nil {
					continue // Skip items that fail to unmarshal
				}
				if !yield(decoded) {
					return
				}
			}
		}
	}
}

// Iter transforms a Result containing collection data into an iterator yielding items of type T.
// This is a convenience method for simple iteration patterns.
//
// IMPORTANT: This panics if the Result has an error or if unmarshaling fails.
// Use Iter2() if you need explicit error handling.
//
// Example:
//
//	result := client.Operations.List(ctx, opts)
//	for op := range op.Iter(result) {
//	    fmt.Printf("Operation: %s\n", op.ID())
//	}
func Iter[T Unwrapper](r Result[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		// Panic if the Result itself has an error
		if r.Err != nil {
			panic("cannot iterate failed result: " + r.Err.Error())
		}

		// Check if Data implements CollectionIterator
		if iterable, ok := any(r.Data).(types.CollectionIterator); ok {
			for item := range iterable.IterBytes() {
				var decoded T
				if err := json.Unmarshal(item.Bytes(), &decoded); err != nil {
					panic("unmarshal error in Iter: " + err.Error())
				}
				if !yield(decoded) {
					return
				}
			}
		}
	}
}

// Iter2 transforms a Result containing collection data into an iterator yielding items and errors.
// This is the error-safe version of Iter() that yields both values and errors.
//
// Use this when you need explicit error handling in your iteration.
//
// Example:
//
//	result := client.Operations.List(ctx, opts)
//	for op, err := range op.Iter2(result) {
//	    if err != nil {
//	        log.Printf("error: %v", err)
//	        continue
//	    }
//	    fmt.Printf("Operation: %s\n", op.ID())
//	}
func Iter2[T Unwrapper](r Result[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// First check if the Result itself has an error
		if r.Err != nil {
			yield(*new(T), r.Err)
			return
		}

		// Check if Data implements CollectionIterator
		if iterable, ok := any(r.Data).(types.CollectionIterator); ok {
			for item := range iterable.IterBytes() {
				var decoded T
				if err := json.Unmarshal(item.Bytes(), &decoded); err != nil {
					// Yield unmarshal errors instead of silently skipping
					if !yield(*new(T), err) {
						return
					}
					continue
				}
				if !yield(decoded, nil) {
					return
				}
			}
		}
	}
}

// IterAsErr transforms an iterator of JSONDoc-based items into an iterator of type U by unmarshaling.
// Unlike IterAs, this yields both the value and any unmarshaling error.
//
// Example:
//
//	for m, err := range collection.IterAsErr[CustomMeasurement]() {
//	    if err != nil {
//	        log.Printf("unmarshal error: %v", err)
//	        continue
//	    }
//	    fmt.Printf("Temp: %.2f\n", m.Temperature.Value)
//	}
func IterAsErr[U any, T Unwrapper](r Result[T]) iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		// Type switch to check if Data has an Iter() method
		type Iterable interface {
			Iter() iter.Seq[T]
		}

		if iterable, ok := any(r.Data).(Iterable); ok {
			for item := range iterable.Iter() {
				var decoded U
				err := json.Unmarshal(item.Bytes(), &decoded)
				if !yield(decoded, err) {
					return
				}
			}
		}
	}
}

// Single converts a Result containing a single item into an iterator that yields
// either the item (if successful) or an error (if the operation failed).
//
// This is designed for pipeline composition where you need to convert single
// operation results into the Seq2[T, error] iterator form.
//
// Example:
//
//	result := client.Operations.Update(ctx, id, data)
//	for op, err := range op.Single(result) {
//	    if err != nil {
//	        log.Printf("update failed: %v", err)
//	        continue
//	    }
//	    fmt.Printf("Updated: %s\n", op.ID())
//	}
func Single[T any](r Result[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if r.Err != nil {
			yield(*new(T), r.Err)
			return
		}
		yield(r.Data, nil)
	}
}

// SingleWithItem converts a Result containing a single item into an iterator,
// preserving the original item when there's an error. This is useful for error
// correlation in pipelines where you need to know which item failed.
//
// When the Result has an error, original is yielded instead of a zero-value.
// This allows OnError callbacks to access the actual item that was being processed.
//
// Example in a pipeline:
//
//	pipeline.Expand(ops, func(operation jsonmodels.Operation) iter.Seq2[jsonmodels.Operation, error] {
//	    result := client.Operations.Update(ctx, operation.ID(), updates)
//	    // If update fails, 'operation' (not zero-value) will be available in OnError
//	    return op.SingleWithItem(operation, result)
//	})
func SingleWithItem[T any](original T, r Result[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if r.Err != nil {
			yield(original, r.Err)
			return
		}
		yield(r.Data, nil)
	}
}

// ToSlice converts an iterator to a slice, collecting all items.
// Returns the slice and the first error encountered (if any).
// Stops iteration on first error.
//
// Example:
//
//	alarms, err := op.ToSlice(alarmCollection.Data.Iter2())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d alarms\n", len(alarms))
func ToSlice[T any](items iter.Seq2[T, error]) ([]T, error) {
	var result []T
	for item, err := range items {
		if err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, nil
}

// ToSliceR converts a Result containing a collection to a slice.
// This is a convenience wrapper around ToSlice for Result types.
//
// Example:
//
//	alarms, err := op.ToSliceR(alarmCollection)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d alarms\n", len(alarms))
func ToSliceR[T Unwrapper](r Result[T]) ([]T, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return ToSlice(Iter2(r))
}

// Collect is an alias for ToSlice for familiarity with other iterator patterns
func Collect[T any](items iter.Seq2[T, error]) ([]T, error) {
	return ToSlice(items)
}

// CollectR is an alias for ToSliceR for familiarity with other iterator patterns
func CollectR[T Unwrapper](r Result[T]) ([]T, error) {
	return ToSliceR(r)
}
