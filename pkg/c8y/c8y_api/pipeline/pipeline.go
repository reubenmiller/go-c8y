package pipeline

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"
	"time"
)

// Options controls the behavior of ForEach and Collect.
type Options struct {
	// Workers is the number of concurrent goroutines processing items.
	// Defaults to 1 (serial execution) when <= 0.
	Workers int

	// Delay is the minimum duration between consecutive items dispatched
	// to each worker. Useful for rate-limiting platform requests.
	Delay time.Duration

	// MaxErrors stops processing after this many errors have occurred.
	// 0 means unlimited — errors are collected but never cause an abort.
	MaxErrors int

	// ErrorThreshold stops processing when the ratio of failed items to
	// total completed items exceeds this value (0.0–1.0).
	// 0 means disabled. For example, 0.1 stops if more than 10% fail.
	ErrorThreshold float64

	// OnProgress is called after each item completes (success or failure).
	// It is called from an unspecified goroutine; the callback must be safe
	// for concurrent use.
	OnProgress func(Stats)

	// OnError is called each time an error occurs (from iterator or user function).
	// It receives the item that failed and the error.
	// The item is passed as interface{} - use type assertion to access its fields.
	// This is called from an unspecified goroutine; the callback must be safe
	// for concurrent use.
	// Example: OnError: func(item any, err error) {
	//     if op, ok := item.(jsonmodels.Operation); ok {
	//         log.Printf("Failed to process %s: %v", op.ID(), err)
	//     }
	// }
	OnError func(item any, err error)
}

func (o Options) workers() int {
	if o.Workers <= 0 {
		return 1
	}
	return o.Workers
}

// Stats holds pipeline execution statistics, reported via OnProgress.
type Stats struct {
	// Completed is the number of items that finished (success + failed).
	Completed int

	// Failed is the number of items that returned a non-nil error.
	Failed int

	// Total is the total number of items seen so far.
	// For streaming sources this may equal Completed if the total is unknown.
	Total int

	// InFlight is the number of items currently being processed by workers.
	InFlight int

	// LastError is the most recent error encountered (from iterator or user function).
	// This is updated each time an error occurs.
	LastError error
}

// ForEach processes each item yielded by items, calling fn for each one.
// It respects context cancellation and the error limits in Options.
//
// Errors from the input sequence are counted toward MaxErrors and ErrorThreshold.
// Processing stops early when:
//   - ctx is cancelled
//   - MaxErrors is reached (if > 0)
//   - ErrorThreshold is exceeded (if > 0)
//
// ForEach returns:
//   - nil if all items completed (even if some failed — use OnProgress to track those)
//   - PipelineError if aborted due to MaxErrors/ErrorThreshold (includes sample errors)
//   - context error if ctx was cancelled
//
// Use Stats.LastError in OnProgress callback to see individual errors as they occur.
func ForEach[T any](
	ctx context.Context,
	items iter.Seq2[T, error],
	opts Options,
	fn func(ctx context.Context, item T) error,
) error {
	_, err := execute(ctx, items, opts, func(ctx context.Context, item T) (struct{}, error) {
		return struct{}{}, fn(ctx, item)
	})
	return err
}

// Batch groups items from the input sequence into slices of up to the given
// size. The last batch may contain fewer items if the input is not evenly
// divisible. Batch is lazy — it yields each batch as soon as enough items
// have been collected.
//
// Errors from the input sequence are yielded immediately without batching.
//
// Example — bulk-create alarms in batches of 100:
//
//	for batch, err := range pipeline.Batch(alarms, 100) {
//	    if err != nil {
//	        log.Printf("error: %v", err)
//	        continue
//	    }
//	    client.Bulk.CreateAlarms(ctx, batch)
//	}
func Batch[T any](items iter.Seq2[T, error], size int) iter.Seq2[[]T, error] {
	if size <= 0 {
		size = 1
	}
	return func(yield func([]T, error) bool) {
		batch := make([]T, 0, size)
		for item, err := range items {
			if err != nil {
				// Yield errors immediately without batching
				if !yield(nil, err) {
					return
				}
				continue
			}

			batch = append(batch, item)
			if len(batch) >= size {
				if !yield(batch, nil) {
					return
				}
				batch = make([]T, 0, size)
			}
		}
		// Yield any remaining items
		if len(batch) > 0 {
			yield(batch, nil)
		}
	}
}

// Throttle limits the rate at which items are yielded from the input sequence.
// It ensures at least the given interval elapses between consecutive items.
// The first item is yielded immediately without delay.
//
// Errors pass through immediately without applying throttling delay.
//
// Use Throttle to prevent overwhelming a rate-limited API:
//
//	// Yield at most 5 items per second
//	throttled := pipeline.Throttle(items, 200 * time.Millisecond)
//	err := pipeline.ForEach(ctx, throttled, pipeline.Options{}, updateFn)
//
// For concurrent workers, use Options.Delay instead — it applies a per-worker
// delay. Throttle applies a global rate limit before items reach the workers.
func Throttle[T any](items iter.Seq2[T, error], interval time.Duration) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		first := true
		for item, err := range items {
			if err != nil {
				// Errors pass through without throttling
				if !yield(*new(T), err) {
					return
				}
				continue
			}

			if !first && interval > 0 {
				time.Sleep(interval)
			}
			first = false
			if !yield(item, nil) {
				return
			}
		}
	}
}

// Concat concatenates multiple sequences into a single sequence.
// Items from the first sequence are yielded COMPLETELY before the second starts.
// This gives you sequential, ordered consumption: seq1[0], seq1[1], ..., seq1[n], seq2[0], ...
//
// Use Concat when you want deterministic ordering by source:
//
//	ops1 := op.Iter(client.Operations.ListAll(ctx, operations.ListOptions{Status: "PENDING"}))
//	ops2 := op.Iter(client.Operations.ListAll(ctx, operations.ListOptions{Status: "EXECUTING"}))
//	allOps := pipeline.Concat(ops1, ops2)  // All PENDING first, then all EXECUTING
//	err := pipeline.ForEach(ctx, allOps, pipeline.Options{}, updateFn)
//
// For interleaved/concurrent consumption from multiple sources, use Merge instead.
func Concat[T any](sequences ...iter.Seq2[T, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, seq := range sequences {
			for item, err := range seq {
				if !yield(item, err) {
					return
				}
			}
		}
	}
}

// Merge concurrently consumes from multiple sequences and yields items as they arrive.
// Unlike Concat, which exhausts the first sequence before starting the second,
// Merge starts all sequences concurrently and yields items in arrival order.
//
// This gives you interleaved, non-deterministic ordering based on API response times:
//
//	ops1 := op.Iter(client.Operations.ListAll(ctx, operations.ListOptions{Status: "PENDING"}))
//	ops2 := op.Iter(client.Operations.ListAll(ctx, operations.ListOptions{Status: "EXECUTING"}))
//	allOps := pipeline.Merge(ops1, ops2)  // Items interleaved as they arrive
//	err := pipeline.ForEach(ctx, allOps, pipeline.Options{}, updateFn)
//
// Merge is useful when you want to start processing items immediately from any source,
// rather than waiting for the first source to complete.
func Merge[T any](sequences ...iter.Seq2[T, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if len(sequences) == 0 {
			return
		}
		if len(sequences) == 1 {
			for item, err := range sequences[0] {
				if !yield(item, err) {
					return
				}
			}
			return
		}

		// Channel to collect items and errors from all sequences
		type result struct {
			item T
			err  error
		}
		items := make(chan result)
		done := make(chan struct{})
		var wg sync.WaitGroup

		// Start a goroutine for each sequence
		for _, seq := range sequences {
			wg.Add(1)
			go func(s iter.Seq2[T, error]) {
				defer wg.Done()
				for item, err := range s {
					select {
					case items <- result{item, err}:
					case <-done:
						return
					}
				}
			}(seq)
		}

		// Close items channel when all sequences are done
		go func() {
			wg.Wait()
			close(items)
		}()

		// Yield items as they arrive
		for r := range items {
			if !yield(r.item, r.err) {
				close(done)
				return
			}
		}
	}
}

// Expand takes each item from the input sequence, calls fn to produce a new
// sequence, and flattens all results into a single iter.Seq[U].
//
// This is the equivalent of a Unix pipe stage — it transforms and expands
// items without nesting. The expansion is lazy and serial: items from the
// inner sequence are yielded before advancing to the next input item.
//
// Since fn is called once per input item, any API calls inside fn fire at
// the rate the input sequence produces items. To throttle those calls, apply
// Throttle to the input sequence before passing it to Expand:
//
//	devices := client.Devices.ListAll(ctx, opts).Items()
//	throttled := pipeline.Throttle(devices, 500 * time.Millisecond)
//	pendingOps := pipeline.Expand(throttled, func(d jsonmodels.ManagedObject) iter.Seq2[jsonmodels.Operation, error] {
//	    return op.Iter(client.Operations.ListAll(ctx, operations.ListOptions{
//	        DeviceID: d.ID(), Status: "PENDING",
//	    }))
//	})
//	err := pipeline.ForEach(ctx, pendingOps, pipeline.Options{Workers: 5}, updateFn)
func Expand[T, U any](items iter.Seq2[T, error], fn func(T) iter.Seq2[U, error]) iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		for item, err := range items {
			if err != nil {
				// Propagate input errors
				if !yield(*new(U), err) {
					return
				}
				continue
			}
			// Expand successful items
			for u, err := range fn(item) {
				if !yield(u, err) {
					return
				}
			}
		}
	}
}

// Empty returns an empty sequence that yields no items.
// Useful for conditional logic in Expand when you want to skip an item.
// Note: Requires explicit type parameter. Use EmptyOf for type inference.
//
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq2[jsonmodels.Operation, error] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.Empty[jsonmodels.Operation]()  // Skip successful operations
//	    }
//	    return op.Single(client.Operations.Update(ctx, op.ID(), body))
//	})
func Empty[T any]() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {}
}

// EmptyOf returns an empty sequence that yields no items.
// The type is inferred from the dummy parameter, avoiding explicit type parameters.
// Useful for conditional logic in Expand when you want to skip an item:
//
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq2[jsonmodels.Operation, error] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.EmptyOf(op)  // Skip - type inferred from op
//	    }
//	    return op.Single(client.Operations.Update(ctx, op.ID(), body))
//	})
func EmptyOf[T any](_ T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {}
}

// Single returns a sequence that yields exactly one item without error.
// Useful for conditional logic in Expand when you want to pass through unchanged:
//
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq2[jsonmodels.Operation, error] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.Single(op)  // Pass through unchanged
//	    }
//	    return op.Single(client.Operations.Update(ctx, op.ID(), body))
//	})
func Single[T any](item T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		yield(item, nil)
	}
}

// Filter returns a new sequence containing only items that satisfy the predicate.
// Items for which predicate returns false are skipped.
//
//	// Only process failed operations
//	failedOps := pipeline.Filter(allOps, func(op jsonmodels.Operation) bool {
//	    return op.Status() == "FAILED"
//	})
//	err := pipeline.ForEach(ctx, failedOps, opts, updateFn)
func Filter[T any](items iter.Seq2[T, error], predicate func(T) bool) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for item, err := range items {
			if err != nil {
				// Always propagate errors
				if !yield(*new(T), err) {
					return
				}
				continue
			}
			if predicate(item) {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}

// Collect processes each item yielded by items, calling fn for each one,
// and returns a slice of all successful results.
//
// Errors from the input sequence are counted toward error limits.
// Results are returned in completion order (not input order) when Workers > 1.
// Error handling follows the same rules as ForEach.
func Collect[T, R any](
	ctx context.Context,
	items iter.Seq2[T, error],
	opts Options,
	fn func(ctx context.Context, item T) (R, error),
) ([]R, error) {
	return execute(ctx, items, opts, fn)
}

// execute is the shared implementation for ForEach and Collect.
func execute[T, R any](
	ctx context.Context,
	items iter.Seq2[T, error],
	opts Options,
	fn func(ctx context.Context, item T) (R, error),
) ([]R, error) {
	numWorkers := opts.workers()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channels
	type workItem struct {
		item T
	}
	work := make(chan workItem, numWorkers)

	type resultItem struct {
		item  T // Original item for error correlation
		value R
		err   error
	}
	results := make(chan resultItem, numWorkers)

	// Counters (accessed atomically from workers, read from collector)
	var completed atomic.Int64
	var failed atomic.Int64
	var inFlight atomic.Int64

	// Error tracking
	var errorsMu sync.Mutex
	var errors []error // Collect first N errors
	const maxErrors = 10
	var lastError error

	// Abort error (set at most once)
	var abortErr error
	var abortOnce sync.Once

	// Signal abort
	abort := func(err error) {
		abortOnce.Do(func() {
			abortErr = err
			cancel()
		})
	}

	// Check if error limits are exceeded
	checkLimits := func() {
		c := completed.Load()
		f := failed.Load()

		if opts.MaxErrors > 0 && int(f) >= opts.MaxErrors {
			abort(&AbortError{
				Reason:    "max errors reached",
				Completed: int(c),
				Failed:    int(f),
			})
			return
		}
		if opts.ErrorThreshold > 0 && c > 0 {
			ratio := float64(f) / float64(c)
			if ratio > opts.ErrorThreshold {
				abort(&AbortError{
					Reason:    "error threshold exceeded",
					Completed: int(c),
					Failed:    int(f),
					Threshold: opts.ErrorThreshold,
				})
			}
		}
	}

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wi := range work {
				inFlight.Add(1)
				value, err := fn(ctx, wi.item)
				inFlight.Add(-1)

				completed.Add(1)
				if err != nil {
					failed.Add(1)
				}

				// Send result (include original item for error correlation)
				select {
				case results <- resultItem{item: wi.item, value: value, err: err}:
				case <-ctx.Done():
					return
				}

				checkLimits()

				// Apply per-worker delay
				if opts.Delay > 0 {
					select {
					case <-time.After(opts.Delay):
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Feed items to workers (in a goroutine so we can collect results concurrently)
	go func() {
		defer close(work)
		for item, err := range items {
			// If the sequence itself has an error, report it as a failure
			if err != nil {
				completed.Add(1)
				failed.Add(1)
				select {
				case results <- resultItem{item: item, value: *new(R), err: err}:
				case <-ctx.Done():
					return
				}
				checkLimits()
				continue
			}

			select {
			case work <- workItem{item: item}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	var collected []R
	for r := range results {
		if r.err == nil {
			collected = append(collected, r.value)
		} else {
			// Track errors
			errorsMu.Lock()
			if len(errors) < maxErrors {
				errors = append(errors, r.err)
			}
			lastError = r.err
			errorsMu.Unlock()

			// Call OnError callback if provided
			if opts.OnError != nil {
				opts.OnError(r.item, r.err)
			}
		}

		if opts.OnProgress != nil {
			c := completed.Load()
			f := failed.Load()
			inf := inFlight.Load()
			errorsMu.Lock()
			lastErr := lastError
			errorsMu.Unlock()
			opts.OnProgress(Stats{
				Completed: int(c),
				Failed:    int(f),
				Total:     int(c) + int(inf),
				InFlight:  int(inf),
				LastError: lastErr,
			})
		}
	}

	if abortErr != nil {
		// Wrap abort error with sample errors
		errorsMu.Lock()
		sampleErrs := make([]error, len(errors))
		copy(sampleErrs, errors)
		errorsMu.Unlock()

		if ae, ok := abortErr.(*AbortError); ok {
			ae.SampleErrors = sampleErrs
		}
		return collected, abortErr
	}

	// If there were errors but we didn't abort, still return them
	errorsMu.Lock()
	hasErrors := len(errors) > 0
	sampleErrs := make([]error, len(errors))
	copy(sampleErrs, errors)
	errorsMu.Unlock()

	if hasErrors {
		c := completed.Load()
		f := failed.Load()
		return collected, &PipelineError{
			Completed:    int(c),
			Failed:       int(f),
			SampleErrors: sampleErrs,
		}
	}

	return collected, ctx.Err()
}

// PipelineError is returned when the pipeline completes with errors
// but was not aborted early. It includes samples of the errors encountered.
type PipelineError struct {
	Completed    int
	Failed       int
	SampleErrors []error // First N errors encountered
}

func (e *PipelineError) Error() string {
	msg := "pipeline completed with errors: " + itoa(e.Failed) + " failed out of " + itoa(e.Completed)
	if len(e.SampleErrors) > 0 {
		msg += " (first error: " + e.SampleErrors[0].Error() + ")"
	}
	return msg
}

// Unwrap returns the first sample error for error chain inspection
func (e *PipelineError) Unwrap() error {
	if len(e.SampleErrors) > 0 {
		return e.SampleErrors[0]
	}
	return nil
}

// AbortError is returned when pipeline processing is stopped early
// due to MaxErrors or ErrorThreshold limits being exceeded.
type AbortError struct {
	Reason       string
	Completed    int
	Failed       int
	Threshold    float64
	SampleErrors []error // First N errors encountered
}

func (e *AbortError) Error() string {
	msg := ""
	if e.Threshold > 0 {
		msg = "pipeline aborted: " + e.Reason +
			" (failed " + itoa(e.Failed) + "/" + itoa(e.Completed) +
			", threshold " + ftoa(e.Threshold) + ")"
	} else {
		msg = "pipeline aborted: " + e.Reason +
			" (failed " + itoa(e.Failed) + "/" + itoa(e.Completed) + ")"
	}
	if len(e.SampleErrors) > 0 {
		msg += " (first error: " + e.SampleErrors[0].Error() + ")"
	}
	return msg
}

// Unwrap returns the first sample error for error chain inspection
func (e *AbortError) Unwrap() error {
	if len(e.SampleErrors) > 0 {
		return e.SampleErrors[0]
	}
	return nil
}

// itoa is a simple int-to-string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	b := make([]byte, 0, 10)
	for n > 0 {
		b = append(b, byte('0'+n%10))
		n /= 10
	}
	if neg {
		b = append(b, '-')
	}
	// reverse
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// ftoa formats a float with 2 decimal places without importing fmt/strconv.
func ftoa(f float64) string {
	whole := int(f)
	frac := int((f - float64(whole)) * 100)
	if frac < 0 {
		frac = -frac
	}
	s := itoa(whole) + "."
	if frac < 10 {
		s += "0"
	}
	s += itoa(frac)
	return s
}
