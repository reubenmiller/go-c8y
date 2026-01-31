package op

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Step represents a composable operation that transforms input T to output T
type Step[T any] func(ctx context.Context, in T) (T, error)

// Pipe chains multiple steps sequentially
// Each step receives the output of the previous step
func Pipe[T any](steps ...Step[T]) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		var err error
		for _, step := range steps {
			select {
			case <-ctx.Done():
				return in, ctx.Err()
			default:
			}

			in, err = step(ctx, in)
			if err != nil {
				return in, err
			}
		}
		return in, nil
	}
}

// Parallel executes multiple steps concurrently on the same input
// All steps receive the same input value
// Returns when all steps complete or first error occurs
func Parallel[T any](steps ...Step[T]) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		if len(steps) == 0 {
			return in, nil
		}

		type result struct {
			value T
			err   error
		}

		results := make(chan result, len(steps))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var wg sync.WaitGroup
		for _, step := range steps {
			wg.Add(1)
			go func(s Step[T]) {
				defer wg.Done()
				val, err := s(ctx, in)
				select {
				case results <- result{value: val, err: err}:
				case <-ctx.Done():
				}
			}(step)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect results - return first error
		var lastResult T = in
		for r := range results {
			if r.err != nil {
				cancel()
				return lastResult, r.err
			}
			lastResult = r.value
		}

		return lastResult, nil
	}
}

// ParallelCollect executes multiple steps concurrently and collects all results
// Returns a slice of results and any errors encountered
func ParallelCollect[T any](steps ...Step[T]) func(context.Context, T) ([]T, []error) {
	return func(ctx context.Context, in T) ([]T, []error) {
		if len(steps) == 0 {
			return nil, nil
		}

		type result struct {
			index int
			value T
			err   error
		}

		results := make(chan result, len(steps))
		var wg sync.WaitGroup

		for i, step := range steps {
			wg.Add(1)
			go func(idx int, s Step[T]) {
				defer wg.Done()
				val, err := s(ctx, in)
				results <- result{index: idx, value: val, err: err}
			}(i, step)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		values := make([]T, len(steps))
		var errs []error

		for r := range results {
			values[r.index] = r.value
			if r.err != nil {
				errs = append(errs, r.err)
			}
		}

		return values, errs
	}
}

// If conditionally executes one of two steps based on a predicate
func If[T any](predicate func(T) bool, thenStep, elseStep Step[T]) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		if predicate(in) {
			return thenStep(ctx, in)
		}
		return elseStep(ctx, in)
	}
}

// IfNil conditionally executes a step only if input is not nil/zero
func IfNil[T any](step Step[T]) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		// For pointer types, check if nil
		// For other types, execute step anyway
		return step(ctx, in)
	}
}

// Map transforms the result of a step using a mapping function
func Map[T, U any](step Step[T], mapper func(T) U) func(context.Context, T) (U, error) {
	return func(ctx context.Context, in T) (U, error) {
		result, err := step(ctx, in)
		if err != nil {
			var zero U
			return zero, err
		}
		return mapper(result), nil
	}
}

// Tap allows side effects without modifying the pipeline value
func Tap[T any](fn func(context.Context, T) error) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		err := fn(ctx, in)
		return in, err
	}
}

// Noop returns the input unchanged - useful for conditional pipelines
func Noop[T any]() Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		return in, nil
	}
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int              // Maximum number of retry attempts (0 = no retry)
	InitialInterval time.Duration    // Initial backoff interval
	MaxInterval     time.Duration    // Maximum backoff interval
	Multiplier      float64          // Backoff multiplier (e.g., 2.0 for exponential)
	Jitter          bool             // Add random jitter to backoff
	RetryableCheck  func(error) bool // Function to determine if error is retryable
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:     3,
	InitialInterval: 100 * time.Millisecond,
	MaxInterval:     5 * time.Second,
	Multiplier:      2.0,
	Jitter:          true,
	RetryableCheck:  nil, // Retry all errors by default
}

// Retry wraps a step with retry logic and exponential backoff
func Retry[T any](step Step[T], config RetryConfig) Step[T] {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.InitialInterval <= 0 {
		config.InitialInterval = 100 * time.Millisecond
	}
	if config.MaxInterval <= 0 {
		config.MaxInterval = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}

	return func(ctx context.Context, in T) (T, error) {
		var lastErr error
		interval := config.InitialInterval

		for attempt := 0; attempt < config.MaxAttempts; attempt++ {
			select {
			case <-ctx.Done():
				return in, ctx.Err()
			default:
			}

			result, err := step(ctx, in)
			if err == nil {
				return result, nil
			}

			lastErr = err

			// Check if error is retryable
			if config.RetryableCheck != nil && !config.RetryableCheck(err) {
				return result, err
			}

			// Don't sleep after last attempt
			if attempt < config.MaxAttempts-1 {
				sleepDuration := calculateBackoff(interval, config.MaxInterval, config.Multiplier, config.Jitter)

				select {
				case <-time.After(sleepDuration):
				case <-ctx.Done():
					return result, ctx.Err()
				}

				interval = time.Duration(float64(interval) * config.Multiplier)
			}
		}

		return in, fmt.Errorf("retry exhausted after %d attempts: %w", config.MaxAttempts, lastErr)
	}
}

// calculateBackoff computes the next backoff duration with optional jitter
func calculateBackoff(current, max time.Duration, multiplier float64, jitter bool) time.Duration {
	next := time.Duration(float64(current) * multiplier)
	if next > max {
		next = max
	}

	if jitter {
		// Add up to 25% random jitter
		jitterAmount := float64(next) * 0.25 * rand.Float64()
		next = time.Duration(float64(next) + jitterAmount)
	}

	return next
}

// Recover allows graceful error recovery with a fallback function
func Recover[T any](step Step[T], fallback func(error) (T, error)) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		result, err := step(ctx, in)
		if err != nil {
			return fallback(err)
		}
		return result, nil
	}
}

// RecoverWith provides a default value on error
func RecoverWith[T any](step Step[T], defaultValue T) Step[T] {
	return Recover(step, func(err error) (T, error) {
		return defaultValue, nil
	})
}

// Timeout wraps a step with a timeout
func Timeout[T any](step Step[T], duration time.Duration) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		ctx, cancel := context.WithTimeout(ctx, duration)
		defer cancel()

		type result struct {
			value T
			err   error
		}

		done := make(chan result, 1)
		go func() {
			val, err := step(ctx, in)
			done <- result{value: val, err: err}
		}()

		select {
		case r := <-done:
			return r.value, r.err
		case <-ctx.Done():
			return in, ctx.Err()
		}
	}
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	mu           sync.RWMutex
	failures     int
	lastFailTime time.Time
	state        CircuitState
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// WithCircuitBreaker wraps a step with circuit breaker logic
func WithCircuitBreaker[T any](step Step[T], cb *CircuitBreaker) Step[T] {
	return func(ctx context.Context, in T) (T, error) {
		if !cb.allowRequest() {
			var zero T
			return zero, ErrCircuitOpen
		}

		result, err := step(ctx, in)

		if err != nil {
			cb.recordFailure()
			return result, err
		}

		cb.recordSuccess()
		return result, nil
	}
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitClosed {
		return true
	}

	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	}

	// Half-open: allow one request through
	return true
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failures = 0
	}
}

// Debounce ensures a step is not executed more frequently than the specified duration
func Debounce[T any](step Step[T], duration time.Duration) Step[T] {
	var (
		mu       sync.Mutex
		lastExec time.Time
		timer    *time.Timer
		pending  bool
	)

	return func(ctx context.Context, in T) (T, error) {
		mu.Lock()
		defer mu.Unlock()

		if time.Since(lastExec) < duration {
			if !pending {
				pending = true
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(duration-time.Since(lastExec), func() {
					mu.Lock()
					pending = false
					mu.Unlock()
				})
			}
			return in, fmt.Errorf("debounced: too frequent execution")
		}

		lastExec = time.Now()
		pending = false

		mu.Unlock()
		result, err := step(ctx, in)
		mu.Lock()

		return result, err
	}
}

// RateLimit limits the execution rate of a step
func RateLimit[T any](step Step[T], requestsPerSecond float64) Step[T] {
	interval := time.Duration(float64(time.Second) / requestsPerSecond)
	ticker := time.NewTicker(interval)

	return func(ctx context.Context, in T) (T, error) {
		select {
		case <-ticker.C:
			return step(ctx, in)
		case <-ctx.Done():
			return in, ctx.Err()
		}
	}
}

// Memoize caches the result of a step based on a key function
func Memoize[T any, K comparable](step Step[T], keyFunc func(T) K) Step[T] {
	cache := make(map[K]T)
	var mu sync.RWMutex

	return func(ctx context.Context, in T) (T, error) {
		key := keyFunc(in)

		mu.RLock()
		if cached, ok := cache[key]; ok {
			mu.RUnlock()
			return cached, nil
		}
		mu.RUnlock()

		result, err := step(ctx, in)
		if err != nil {
			return result, err
		}

		mu.Lock()
		cache[key] = result
		mu.Unlock()

		return result, nil
	}
}
