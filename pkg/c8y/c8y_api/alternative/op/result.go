package op

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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

// WithMeta adds metadata to the result
func (r Result[T]) WithMeta(key string, value any) Result[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[key] = value
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

func First[T any](items iter.Seq[T]) (Result[T], bool) {
	for item := range items {
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
func IterAs[U any, T Unwrapper](r Result[T]) iter.Seq[*U] {
	return func(yield func(*U) bool) {
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
				if !yield(&decoded) {
					return
				}
			}
		}
	}
}

func Iter[T Unwrapper](r Result[T]) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		// Check if Data implements CollectionIterator
		if iterable, ok := any(r.Data).(types.CollectionIterator); ok {
			for item := range iterable.IterBytes() {
				decoded := new(T)
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
func IterAsErr[U any, T Unwrapper](r Result[T]) iter.Seq2[*U, error] {
	return func(yield func(*U, error) bool) {
		// Type switch to check if Data has an Iter() method
		type Iterable interface {
			Iter() iter.Seq[T]
		}

		if iterable, ok := any(r.Data).(Iterable); ok {
			for item := range iterable.Iter() {
				var decoded U
				err := json.Unmarshal(item.Bytes(), &decoded)
				if !yield(&decoded, err) {
					return
				}
			}
		}
	}
}
