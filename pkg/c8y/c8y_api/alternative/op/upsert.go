package op

import (
	"context"
	"errors"
	"reflect"
	"time"
)

// UpdateFunc updates an existing resource
type UpdateFunc[T any] func(ctx context.Context, obj T) (T, error)

// MergeFunc merges desired state into existing resource
type MergeFunc[T any] func(existing, desired T) T

// Upsert performs get → update or create operation
// Returns Result with StatusUpdated if updated, StatusCreated if created, StatusSkipped if no changes
func Upsert[T any](
	getter GetFunc[T],
	updater UpdateFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
) func(context.Context, T) (Result[T], error) {
	return func(ctx context.Context, obj T) (Result[T], error) {
		start := time.Now()
		key := keyFunc(obj)

		// Try to get existing resource
		_, err := getter(ctx, key)
		if err == nil {
			// Resource exists, update it
			updated, err := updater(ctx, obj)
			if err != nil {
				return Failed[T](err, isRetryableError(err)), err
			}

			return Updated(updated, map[string]any{
				"key": key,
			}).WithDuration(time.Since(start)), nil
		}

		// If error is not "not found", return it
		if !errors.Is(err, ErrNotFound) {
			return Failed[T](err, isRetryableError(err)), err
		}

		// Resource doesn't exist, create it
		created, err := creator(ctx, obj)
		if err != nil {
			return Failed[T](err, isRetryableError(err)), err
		}

		return Created(created, map[string]any{
			"key": key,
		}).WithDuration(time.Since(start)), nil
	}
}

// UpsertWithMerge performs upsert with delta calculation
// Only updates if changes detected between existing and desired state
func UpsertWithMerge[T any](
	getter GetFunc[T],
	updater UpdateFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
	mergeFunc MergeFunc[T],
) func(context.Context, T) (Result[T], error) {
	return func(ctx context.Context, desired T) (Result[T], error) {
		start := time.Now()
		key := keyFunc(desired)

		// Try to get existing resource
		existing, err := getter(ctx, key)
		if err == nil {
			// Merge desired changes into existing
			merged := mergeFunc(existing, desired)

			// Check if there are actual changes
			if reflect.DeepEqual(existing, merged) {
				return Skipped(existing, "no changes detected").
					WithDuration(time.Since(start)), nil
			}

			// Update with merged data
			updated, err := updater(ctx, merged)
			if err != nil {
				return Failed[T](err, isRetryableError(err)), err
			}

			return Updated(updated, map[string]any{
				"key":     key,
				"changed": true,
			}).WithDuration(time.Since(start)), nil
		}

		// If error is not "not found", return it
		if !errors.Is(err, ErrNotFound) {
			return Failed[T](err, isRetryableError(err)), err
		}

		// Resource doesn't exist, create it
		created, err := creator(ctx, desired)
		if err != nil {
			return Failed[T](err, isRetryableError(err)), err
		}

		return Created(created, map[string]any{
			"key": key,
		}).WithDuration(time.Since(start)), nil
	}
}

// UpsertWithConflictResolution handles 409 Conflict responses
// Retries with the updated resource on conflict
func UpsertWithConflictResolution[T any](
	getter GetFunc[T],
	updater UpdateFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
	maxConflictRetries int,
) func(context.Context, T) (Result[T], error) {
	if maxConflictRetries <= 0 {
		maxConflictRetries = 3
	}

	return func(ctx context.Context, obj T) (Result[T], error) {
		start := time.Now()
		var lastResult Result[T]
		var lastErr error

		for attempt := 0; attempt < maxConflictRetries; attempt++ {
			key := keyFunc(obj)

			// Try to get existing resource
			_, err := getter(ctx, key)
			if err == nil {
				// Resource exists, update it
				updated, err := updater(ctx, obj)
				if err != nil {
					// Check if it's a conflict error
					if isConflictError(err) && attempt < maxConflictRetries-1 {
						// Retry with fresh data
						lastErr = err
						time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
						continue
					}
					return Failed[T](err, isRetryableError(err)), err
				}

				return Updated(updated, map[string]any{
					"key":      key,
					"attempts": attempt + 1,
				}).WithDuration(time.Since(start)).WithAttempts(attempt + 1), nil
			}

			// If error is not "not found", check if retryable
			if !errors.Is(err, ErrNotFound) {
				if isRetryableError(err) && attempt < maxConflictRetries-1 {
					lastErr = err
					time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
					continue
				}
				return Failed[T](err, isRetryableError(err)), err
			}

			// Resource doesn't exist, create it
			created, err := creator(ctx, obj)
			if err != nil {
				// Check if another process created it (409)
				if isConflictError(err) && attempt < maxConflictRetries-1 {
					// Retry - it should exist now
					lastErr = err
					time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
					continue
				}
				return Failed[T](err, isRetryableError(err)), err
			}

			return Created(created, map[string]any{
				"key":      key,
				"attempts": attempt + 1,
			}).WithDuration(time.Since(start)).WithAttempts(attempt + 1), nil
		}

		return lastResult, lastErr
	}
}

// UpsertBatch performs batch upsert operations
func UpsertBatch[T any](
	getter GetFunc[T],
	updater UpdateFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
) func(context.Context, []T) ([]Result[T], error) {
	upsertFn := Upsert(getter, updater, creator, keyFunc)

	return func(ctx context.Context, objects []T) ([]Result[T], error) {
		results := make([]Result[T], len(objects))
		var firstErr error

		for i, obj := range objects {
			result, err := upsertFn(ctx, obj)
			results[i] = result

			if err != nil && firstErr == nil {
				firstErr = err
			}

			// Check context cancellation
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}
		}

		return results, firstErr
	}
}

// UpsertWithOptimisticLocking uses version/etag for concurrency control
func UpsertWithOptimisticLocking[T any](
	getter GetFunc[T],
	updater UpdateFunc[T],
	creator CreateFunc[T],
	keyFunc KeyFunc[T],
	versionFunc func(T) string,
	maxRetries int,
) func(context.Context, T) (Result[T], error) {
	if maxRetries <= 0 {
		maxRetries = 5
	}

	return func(ctx context.Context, obj T) (Result[T], error) {
		start := time.Now()

		for attempt := 0; attempt < maxRetries; attempt++ {
			key := keyFunc(obj)

			// Get latest version
			existing, err := getter(ctx, key)
			if err == nil {
				// Verify version matches (if version is set on obj)
				objVersion := versionFunc(obj)
				existingVersion := versionFunc(existing)

				if objVersion != "" && objVersion != existingVersion {
					// Version mismatch - someone else updated it
					if attempt < maxRetries-1 {
						// Retry with fresh data
						time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
						continue
					}
					return Failed[T](errors.New("version conflict after max retries"), false),
						errors.New("version conflict")
				}

				// Update
				updated, err := updater(ctx, obj)
				if err != nil {
					if isConflictError(err) && attempt < maxRetries-1 {
						time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
						continue
					}
					return Failed[T](err, isRetryableError(err)), err
				}

				return Updated(updated, map[string]any{
					"key":      key,
					"version":  versionFunc(updated),
					"attempts": attempt + 1,
				}).WithDuration(time.Since(start)).WithAttempts(attempt + 1), nil
			}

			// If not found, create
			if errors.Is(err, ErrNotFound) {
				created, err := creator(ctx, obj)
				if err != nil {
					if isConflictError(err) && attempt < maxRetries-1 {
						time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
						continue
					}
					return Failed[T](err, isRetryableError(err)), err
				}

				return Created(created, map[string]any{
					"key":      key,
					"version":  versionFunc(created),
					"attempts": attempt + 1,
				}).WithDuration(time.Since(start)).WithAttempts(attempt + 1), nil
			}

			// Other error
			if isRetryableError(err) && attempt < maxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}

			return Failed[T](err, isRetryableError(err)), err
		}

		return Failed[T](errors.New("max retries exceeded"), false),
			errors.New("max retries exceeded")
	}
}

// isConflictError checks if error is a 409 Conflict
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "409") || contains(errStr, "conflict")
}

// DeltaMerge provides a generic merge function that only updates non-zero fields
func DeltaMerge[T any](existing, desired T) T {
	// Use reflection to merge only non-zero fields from desired into existing
	existingVal := reflect.ValueOf(&existing).Elem()
	desiredVal := reflect.ValueOf(desired)

	mergeFields(existingVal, desiredVal)

	return existing
}

// mergeFields recursively merges non-zero fields
func mergeFields(dst, src reflect.Value) {
	if !dst.CanSet() {
		return
	}

	switch src.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			srcField := src.Field(i)
			dstField := dst.Field(i)

			if dstField.CanSet() && !isZero(srcField) {
				mergeFields(dstField, srcField)
			}
		}
	case reflect.Ptr:
		if !src.IsNil() {
			if dst.IsNil() {
				dst.Set(reflect.New(src.Elem().Type()))
			}
			mergeFields(dst.Elem(), src.Elem())
		}
	case reflect.Map:
		if !src.IsNil() && src.Len() > 0 {
			if dst.IsNil() {
				dst.Set(reflect.MakeMap(src.Type()))
			}
			for _, key := range src.MapKeys() {
				dst.SetMapIndex(key, src.MapIndex(key))
			}
		}
	case reflect.Slice:
		if !src.IsNil() && src.Len() > 0 {
			dst.Set(src)
		}
	default:
		if !isZero(src) {
			dst.Set(src)
		}
	}
}

// isZero checks if a value is the zero value for its type
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return v.Interface() == reflect.Zero(v.Type()).Interface()
	}
}
