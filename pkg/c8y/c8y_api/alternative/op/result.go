package op

import (
	"time"
)

// Status represents the outcome of an operation
type Status string

const (
	StatusOK      Status = "OK"      // Existing resource retrieved
	StatusCreated Status = "Created" // New resource created
	StatusUpdated Status = "Updated" // Existing resource modified
	StatusSkipped Status = "Skipped" // Operation skipped (e.g., no-op update)
	StatusFailed  Status = "Failed"  // Operation failed
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
}

func Ok[T any](v T) Result[T] {
	return Result[T]{Data: v, Status: StatusOK}
}

func Created[T any](v T) Result[T] {
	return Result[T]{Data: v, Status: StatusCreated}
}

func Failed[T any](err error, retryable bool) Result[T] {
	return Result[T]{Err: err, Status: StatusFailed, Retryable: retryable}
}

// NewOK creates a successful result for retrieved resource
func NewOK[T any](data T, meta ...map[string]any) Result[T] {
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

// NewCreated creates a result for newly created resource
func NewCreated[T any](data T, meta ...map[string]any) Result[T] {
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

// NewUpdated creates a result for updated resource
func NewUpdated[T any](data T, meta ...map[string]any) Result[T] {
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

// NewSkipped creates a result for skipped operation
func NewSkipped[T any](data T, reason string) Result[T] {
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

// NewFailed creates a failed result with error
func NewFailed[T any](err error, retryable bool) Result[T] {
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
