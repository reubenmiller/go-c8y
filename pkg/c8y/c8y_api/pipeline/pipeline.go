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
}

// ForEach processes each item yielded by items, calling fn for each one.
// It respects context cancellation and the error limits in Options.
//
// Processing stops early when:
//   - ctx is cancelled
//   - MaxErrors is reached (if > 0)
//   - ErrorThreshold is exceeded (if > 0)
//
// ForEach returns the first error that caused an abort, or nil if all items
// completed (even if some individual items failed — use OnProgress to track those).
// If the pipeline is aborted due to error limits, the returned error wraps
// the most recent item error.
func ForEach[T any](
	ctx context.Context,
	items iter.Seq[T],
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
// Example — bulk-create alarms in batches of 100:
//
//	for batch := range pipeline.Batch(alarms, 100) {
//	    client.Bulk.CreateAlarms(ctx, batch)
//	}
func Batch[T any](items iter.Seq[T], size int) iter.Seq[[]T] {
	if size <= 0 {
		size = 1
	}
	return func(yield func([]T) bool) {
		batch := make([]T, 0, size)
		for item := range items {
			batch = append(batch, item)
			if len(batch) >= size {
				if !yield(batch) {
					return
				}
				batch = make([]T, 0, size)
			}
		}
		// Yield any remaining items
		if len(batch) > 0 {
			yield(batch)
		}
	}
}

// Throttle limits the rate at which items are yielded from the input sequence.
// It ensures at least the given interval elapses between consecutive items.
// The first item is yielded immediately without delay.
//
// Use Throttle to prevent overwhelming a rate-limited API:
//
//	// Yield at most 5 items per second
//	throttled := pipeline.Throttle(items, 200 * time.Millisecond)
//	err := pipeline.ForEach(ctx, throttled, pipeline.Options{}, updateFn)
//
// For concurrent workers, use Options.Delay instead — it applies a per-worker
// delay. Throttle applies a global rate limit before items reach the workers.
func Throttle[T any](items iter.Seq[T], interval time.Duration) iter.Seq[T] {
	return func(yield func(T) bool) {
		first := true
		for item := range items {
			if !first && interval > 0 {
				time.Sleep(interval)
			}
			first = false
			if !yield(item) {
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
//	ops1 := client.Operations.ListAll(ctx, operations.ListOptions{Status: "PENDING"}).Items()
//	ops2 := client.Operations.ListAll(ctx, operations.ListOptions{Status: "EXECUTING"}).Items()
//	allOps := pipeline.Concat(ops1, ops2)  // All PENDING first, then all EXECUTING
//	err := pipeline.ForEach(ctx, allOps, pipeline.Options{}, updateFn)
//
// For interleaved/concurrent consumption from multiple sources, use Merge instead.
func Concat[T any](sequences ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, seq := range sequences {
			for item := range seq {
				if !yield(item) {
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
//	ops1 := client.Operations.ListAll(ctx, operations.ListOptions{Status: "PENDING"}).Items()
//	ops2 := client.Operations.ListAll(ctx, operations.ListOptions{Status: "EXECUTING"}).Items()
//	allOps := pipeline.Merge(ops1, ops2)  // Items interleaved as they arrive
//	err := pipeline.ForEach(ctx, allOps, pipeline.Options{}, updateFn)
//
// Merge is useful when you want to start processing items immediately from any source,
// rather than waiting for the first source to complete.
func Merge[T any](sequences ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		if len(sequences) == 0 {
			return
		}
		if len(sequences) == 1 {
			for item := range sequences[0] {
				if !yield(item) {
					return
				}
			}
			return
		}

		// Channel to collect items from all sequences
		items := make(chan T)
		done := make(chan struct{})
		var wg sync.WaitGroup

		// Start a goroutine for each sequence
		for _, seq := range sequences {
			wg.Add(1)
			go func(s iter.Seq[T]) {
				defer wg.Done()
				for item := range s {
					select {
					case items <- item:
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
		for item := range items {
			if !yield(item) {
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
//	pendingOps := pipeline.Expand(throttled, func(d jsonmodels.ManagedObject) iter.Seq[jsonmodels.Operation] {
//	    return client.Operations.ListAll(ctx, operations.ListOptions{
//	        DeviceID: d.ID(), Status: "PENDING",
//	    }).Items()
//	})
//	err := pipeline.ForEach(ctx, pendingOps, pipeline.Options{Workers: 5}, updateFn)
func Expand[T, U any](items iter.Seq[T], fn func(T) iter.Seq[U]) iter.Seq[U] {
	return func(yield func(U) bool) {
		for item := range items {
			for u := range fn(item) {
				if !yield(u) {
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
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq[jsonmodels.Operation] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.Empty[jsonmodels.Operation]()  // Skip successful operations
//	    }
//	    item := client.Operations.Update(ctx, op.ID(), body)
//	    return op.Iter(item)
//	})
func Empty[T any]() iter.Seq[T] {
	return func(yield func(T) bool) {}
}

// EmptyOf returns an empty sequence that yields no items.
// The type is inferred from the dummy parameter, avoiding explicit type parameters.
// Useful for conditional logic in Expand when you want to skip an item:
//
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq[jsonmodels.Operation] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.EmptyOf(op)  // Skip - type inferred from op
//	    }
//	    item := client.Operations.Update(ctx, op.ID(), body)
//	    return op.Iter(item)
//	})
func EmptyOf[T any](_ T) iter.Seq[T] {
	return func(yield func(T) bool) {}
}

// Single returns a sequence that yields exactly one item.
// Useful for conditional logic in Expand when you want to pass through unchanged:
//
//	result := pipeline.Expand(ops, func(op jsonmodels.Operation) iter.Seq[jsonmodels.Operation] {
//	    if op.Status() == "SUCCESSFUL" {
//	        return pipeline.Single(op)  // Pass through unchanged
//	    }
//	    item := client.Operations.Update(ctx, op.ID(), body)
//	    return op.Iter(item)
//	})
func Single[T any](item T) iter.Seq[T] {
	return func(yield func(T) bool) {
		yield(item)
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
func Filter[T any](items iter.Seq[T], predicate func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range items {
			if predicate(item) {
				if !yield(item) {
					return
				}
			}
		}
	}
}

// Collect processes each item yielded by items, calling fn for each one,
// and returns a slice of all successful results.
//
// Results are returned in completion order (not input order) when Workers > 1.
// Error handling follows the same rules as ForEach.
func Collect[T, R any](
	ctx context.Context,
	items iter.Seq[T],
	opts Options,
	fn func(ctx context.Context, item T) (R, error),
) ([]R, error) {
	return execute(ctx, items, opts, fn)
}

// execute is the shared implementation for ForEach and Collect.
func execute[T, R any](
	ctx context.Context,
	items iter.Seq[T],
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
		value R
		err   error
	}
	results := make(chan resultItem, numWorkers)

	// Counters (accessed atomically from workers, read from collector)
	var completed atomic.Int64
	var failed atomic.Int64
	var inFlight atomic.Int64

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

				// Send result (even on error, for Collect to know about it)
				select {
				case results <- resultItem{value: value, err: err}:
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
		for item := range items {
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
		}

		if opts.OnProgress != nil {
			c := completed.Load()
			f := failed.Load()
			inf := inFlight.Load()
			opts.OnProgress(Stats{
				Completed: int(c),
				Failed:    int(f),
				Total:     int(c) + int(inf),
				InFlight:  int(inf),
			})
		}
	}

	if abortErr != nil {
		return collected, abortErr
	}

	return collected, ctx.Err()
}

// AbortError is returned when pipeline processing is stopped early
// due to MaxErrors or ErrorThreshold limits being exceeded.
type AbortError struct {
	Reason    string
	Completed int
	Failed    int
	Threshold float64
}

func (e *AbortError) Error() string {
	if e.Threshold > 0 {
		return "pipeline aborted: " + e.Reason +
			" (failed " + itoa(e.Failed) + "/" + itoa(e.Completed) +
			", threshold " + ftoa(e.Threshold) + ")"
	}
	return "pipeline aborted: " + e.Reason +
		" (failed " + itoa(e.Failed) + "/" + itoa(e.Completed) + ")"
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
