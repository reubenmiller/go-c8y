package op

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("resource not found")

// GetFunc retrieves a resource by key
type GetFunc[T any] func(ctx context.Context, key string) (T, error)

// CreateFunc creates a new resource
type CreateFunc[T any] func(ctx context.Context, obj T) (T, error)

// FindFunc searches for resources matching a predicate
type FindFunc[T any] func(ctx context.Context, predicate func(T) bool) ([]T, error)

// KeyFunc extracts a key from a resource
type KeyFunc[T any] func(T) string

// GetOrCreate retrieves a resource or creates it if not found
// Returns Result with StatusOK if found, StatusCreated if created
func GetOrCreate[T any](
	getter GetFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
) func(context.Context, T) (Result[T], error) {
	return func(ctx context.Context, obj T) (Result[T], error) {
		start := time.Now()
		key := keyFunc(obj)

		// Try to get existing resource
		existing, err := getter(ctx, key)
		if err == nil {
			return NewOK(existing, map[string]any{
				"key":    key,
				"cached": false,
			}).WithDuration(time.Since(start)), nil
		}

		// If error is not "not found", return it
		if !errors.Is(err, ErrNotFound) {
			return NewFailed[T](err, isRetryableError(err)), err
		}

		// Resource doesn't exist, create it
		created, err := creator(ctx, obj)
		if err != nil {
			return NewFailed[T](err, isRetryableError(err)), err
		}

		return NewCreated(created, map[string]any{
			"key": key,
		}).WithDuration(time.Since(start)), nil
	}
}

// GetOrCreateWithFind uses a find operation instead of get by key
// Useful when searching by non-unique criteria
func GetOrCreateWithFind[T any](
	finder FindFunc[T],
	creator CreateFunc[T],
	matcher func(T) bool,
) func(context.Context, T) (Result[T], error) {
	return func(ctx context.Context, obj T) (Result[T], error) {
		start := time.Now()

		// Try to find matching resource
		found, err := finder(ctx, matcher)
		if err != nil {
			return NewFailed[T](err, isRetryableError(err)), err
		}

		// If found, return first match
		if len(found) > 0 {
			return NewOK(found[0], map[string]any{
				"matchCount": len(found),
			}).WithDuration(time.Since(start)), nil
		}

		// Not found, create it
		created, err := creator(ctx, obj)
		if err != nil {
			return NewFailed[T](err, isRetryableError(err)), err
		}

		return NewCreated(created).WithDuration(time.Since(start)), nil
	}
}

// GetOrCreateIdempotent wraps GetOrCreate with idempotency guarantees
// Uses idempotency key to prevent duplicate creates on retry
func GetOrCreateIdempotent[T any](
	getter GetFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
	idempotencyKey string,
) func(context.Context, T) (Result[T], error) {
	return func(ctx context.Context, obj T) (Result[T], error) {
		// Add idempotency key to context if provided
		if idempotencyKey != "" {
			// Store in context for use by HTTP layer
			ctx = context.WithValue(ctx, "X-Idempotency-Key", idempotencyKey)
		}

		fn := GetOrCreate(getter, creator, keyFunc)
		result, err := fn(ctx, obj)

		if err == nil {
			result.Idempotent = true
		}

		return result, err
	}
}

// TryGetOrCreate attempts get-or-create with retry logic
func TryGetOrCreate[T any](
	getter GetFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
	retryConfig RetryConfig,
) func(context.Context, T) (Result[T], error) {
	baseFunc := GetOrCreate(getter, creator, keyFunc)

	return func(ctx context.Context, obj T) (Result[T], error) {
		var lastResult Result[T]
		var lastErr error
		interval := retryConfig.InitialInterval

		for attempt := 0; attempt < retryConfig.MaxAttempts; attempt++ {
			result, err := baseFunc(ctx, obj)

			if err == nil {
				return result.WithAttempts(attempt + 1), nil
			}

			lastResult = result
			lastErr = err

			// Check if retryable
			if !result.Retryable {
				return result, err
			}

			// Wait before retry
			if attempt < retryConfig.MaxAttempts-1 {
				sleepDuration := calculateBackoff(interval, retryConfig.MaxInterval, retryConfig.Multiplier, retryConfig.Jitter)

				select {
				case <-time.After(sleepDuration):
				case <-ctx.Done():
					return lastResult, ctx.Err()
				}

				interval = time.Duration(float64(interval) * retryConfig.Multiplier)
			}
		}

		return lastResult.WithAttempts(retryConfig.MaxAttempts), lastErr
	}
}

// isRetryableError determines if an error can be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to check for specific retryable errors
	// For now, consider network errors and 5xx status codes retryable
	errStr := err.Error()

	// Network/timeout errors
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return false // Don't retry cancelled/timeout contexts
	}

	// Check for common retryable patterns
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"503",
		"502",
		"500",
		"429", // Rate limit
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
