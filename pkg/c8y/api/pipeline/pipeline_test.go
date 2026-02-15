package pipeline

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachSerial(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	var mu sync.Mutex
	var processed []int
	err := ForEach(context.Background(), items, Options{}, func(_ context.Context, item int) error {
		mu.Lock()
		processed = append(processed, item)
		mu.Unlock()
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, processed, "should process all items in order when serial")
}

func TestForEachConcurrent(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	var count atomic.Int64
	err := ForEach(context.Background(), items, Options{Workers: 3}, func(_ context.Context, _ int) error {
		count.Add(1)
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int64(10), count.Load(), "should process all items")
}

func TestForEachContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	var count atomic.Int64
	err := ForEach(ctx, items, Options{Workers: 1}, func(_ context.Context, item int) error {
		count.Add(1)
		if item == 3 {
			cancel()
		}
		return nil
	})

	assert.Error(t, err)
	assert.True(t, count.Load() <= 5, "should stop processing after context cancellation")
}

func TestForEachMaxErrors(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
	errBoom := errors.New("boom")

	err := ForEach(context.Background(), items, Options{
		Workers:   1,
		MaxErrors: 3,
	}, func(_ context.Context, _ int) error {
		return errBoom
	})

	require.Error(t, err)
	var abortErr *AbortError
	assert.True(t, errors.As(err, &abortErr), "should be an AbortError")
	assert.Equal(t, "max errors reached", abortErr.Reason)
	assert.Equal(t, 3, abortErr.Failed)
}

func TestForEachErrorThreshold(t *testing.T) {
	// Items: 10 total. Fail every other one.
	// After 2 items: 1 fail / 2 completed = 0.5 > 0.3 threshold → abort.
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	err := ForEach(context.Background(), items, Options{
		Workers:        1,
		ErrorThreshold: 0.3,
	}, func(_ context.Context, item int) error {
		if item%2 == 0 {
			return errors.New("even number")
		}
		return nil
	})

	require.Error(t, err)
	var abortErr *AbortError
	assert.True(t, errors.As(err, &abortErr), "should be an AbortError")
	assert.Equal(t, "error threshold exceeded", abortErr.Reason)
}

func TestForEachDelay(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3}))
	start := time.Now()

	err := ForEach(context.Background(), items, Options{
		Workers: 1,
		Delay:   50 * time.Millisecond,
	}, func(_ context.Context, _ int) error {
		return nil
	})

	elapsed := time.Since(start)
	require.NoError(t, err)
	// 3 items with 50ms delay each → at least ~100ms (delay is after processing, not before first)
	assert.True(t, elapsed >= 100*time.Millisecond, "should respect delay between items, elapsed: %v", elapsed)
}

func TestForEachProgress(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	var mu sync.Mutex
	var progressCalls []Stats
	err := ForEach(context.Background(), items, Options{
		Workers: 1,
		OnProgress: func(s Stats) {
			mu.Lock()
			progressCalls = append(progressCalls, s)
			mu.Unlock()
		},
	}, func(_ context.Context, _ int) error {
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, progressCalls, 5, "should call progress for each item")

	// Last call should show 5 completed
	last := progressCalls[len(progressCalls)-1]
	assert.Equal(t, 5, last.Completed)
	assert.Equal(t, 0, last.Failed)
}

func TestForEachProgressWithErrors(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	var mu sync.Mutex
	var lastStats Stats
	err := ForEach(context.Background(), items, Options{
		Workers: 1,
		OnProgress: func(s Stats) {
			mu.Lock()
			lastStats = s
			mu.Unlock()
		},
	}, func(_ context.Context, item int) error {
		if item%2 == 0 {
			return errors.New("even")
		}
		return nil
	})

	// Should complete (not abort) but return PipelineError indicating some items failed
	require.Error(t, err, "should return error when items fail")
	var abortErr *AbortError
	assert.False(t, errors.As(err, &abortErr), "should not abort when no error limits are set")
	var pipelineErr *PipelineError
	assert.True(t, errors.As(err, &pipelineErr), "should return PipelineError")
	assert.Equal(t, 5, lastStats.Completed)
	assert.Equal(t, 2, lastStats.Failed)
}

func TestForEachEmptyInput(t *testing.T) {
	items := Values(slices.Values([]int{}))

	err := ForEach(context.Background(), items, Options{Workers: 3}, func(_ context.Context, _ int) error {
		t.Fatal("should not be called")
		return nil
	})

	assert.NoError(t, err)
}

func TestCollectSerial(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	results, err := Collect(context.Background(), items, Options{}, func(_ context.Context, item int) (int, error) {
		return item * 2, nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results, "should collect all doubled values")
}

func TestCollectConcurrent(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	results, err := Collect(context.Background(), items, Options{Workers: 3}, func(_ context.Context, item int) (string, error) {
		return itoa(item), nil
	})

	require.NoError(t, err)
	assert.Len(t, results, 5, "should collect all results")
	// With concurrent workers, order is not guaranteed
	slices.Sort(results)
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, results)
}

func TestCollectSkipsErrors(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	results, err := Collect(context.Background(), items, Options{Workers: 1}, func(_ context.Context, item int) (int, error) {
		if item%2 == 0 {
			return 0, errors.New("even")
		}
		return item, nil
	})

	// Should complete (not abort) but return PipelineError indicating some items failed
	require.Error(t, err, "should return error when items fail")
	var abortErr *AbortError
	assert.False(t, errors.As(err, &abortErr), "should not abort without error limits")
	var pipelineErr *PipelineError
	assert.True(t, errors.As(err, &pipelineErr), "should return PipelineError")
	assert.Equal(t, []int{1, 3, 5}, results, "should only collect successful results")
}

func TestCollectMaxErrors(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	results, err := Collect(context.Background(), items, Options{
		Workers:   1,
		MaxErrors: 2,
	}, func(_ context.Context, item int) (int, error) {
		if item%2 == 0 {
			return 0, errors.New("even")
		}
		return item, nil
	})

	require.Error(t, err)
	var abortErr *AbortError
	assert.True(t, errors.As(err, &abortErr))
	// Should still return the successful results collected before abort
	assert.NotEmpty(t, results, "should return partial results on abort")
}

func TestCollectEmptyInput(t *testing.T) {
	items := Values(slices.Values([]string{}))

	results, err := Collect(context.Background(), items, Options{}, func(_ context.Context, _ string) (int, error) {
		t.Fatal("should not be called")
		return 0, nil
	})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestForEachConcurrentWorkersActuallyParallel(t *testing.T) {
	// Verify that with multiple workers, items are processed in parallel
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6}))
	var maxConcurrent atomic.Int64
	var current atomic.Int64

	err := ForEach(context.Background(), items, Options{Workers: 3}, func(_ context.Context, _ int) error {
		c := current.Add(1)
		// Track peak concurrency
		for {
			old := maxConcurrent.Load()
			if c <= old || maxConcurrent.CompareAndSwap(old, c) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		current.Add(-1)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, maxConcurrent.Load() > 1, "should have processed items concurrently, max concurrent: %d", maxConcurrent.Load())
}

func TestAbortErrorMessage(t *testing.T) {
	t.Run("with threshold", func(t *testing.T) {
		err := &AbortError{
			Reason:    "error threshold exceeded",
			Completed: 100,
			Failed:    20,
			Threshold: 0.1,
		}
		assert.Contains(t, err.Error(), "error threshold exceeded")
		assert.Contains(t, err.Error(), "20/100")
		assert.Contains(t, err.Error(), "0.10")
	})

	t.Run("without threshold", func(t *testing.T) {
		err := &AbortError{
			Reason:    "max errors reached",
			Completed: 50,
			Failed:    10,
		}
		assert.Contains(t, err.Error(), "max errors reached")
		assert.Contains(t, err.Error(), "10/50")
		assert.NotContains(t, err.Error(), "threshold")
	})
}

func TestForEachWithStrings(t *testing.T) {
	// Simulate the go-c8y-cli pattern: process device names
	names := []string{"MQTT Device 01", "MQTT Device 02", "MQTT Device 03"}
	items := Values(slices.Values(names))

	var mu sync.Mutex
	var updated []string
	err := ForEach(context.Background(), items, Options{Workers: 2}, func(_ context.Context, name string) error {
		mu.Lock()
		updated = append(updated, name)
		mu.Unlock()
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, updated, 3)
}

func TestExpandBasic(t *testing.T) {
	// Expand each number into a sequence of its multiples
	items := Values(slices.Values([]int{1, 2, 3}))
	expanded := Expand(items, func(n int) iter.Seq2[int, error] {
		return Values(slices.Values([]int{n * 10, n * 100}))
	})

	var result []int
	for v := range expanded {
		result = append(result, v)
	}

	assert.Equal(t, []int{10, 100, 20, 200, 30, 300}, result)
}

func TestExpandEmpty(t *testing.T) {
	items := Values(slices.Values([]int{}))
	expanded := Expand(items, func(n int) iter.Seq2[string, error] {
		t.Fatal("should not be called")
		return Values(slices.Values([]string{}))
	})

	var count int
	for range expanded {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestExpandSomeEmpty(t *testing.T) {
	// Some items expand to empty sequences
	items := Values(slices.Values([]int{1, 2, 3}))
	expanded := Expand(items, func(n int) iter.Seq2[int, error] {
		if n == 2 {
			return Values(slices.Values([]int{})) // device with no operations
		}
		return Values(slices.Values([]int{n}))
	})

	var result []int
	for v := range expanded {
		result = append(result, v)
	}

	assert.Equal(t, []int{1, 3}, result)
}

func TestExpandEarlyBreak(t *testing.T) {
	// Consumer stops early — Expand should stop too
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))
	expanded := Expand(items, func(n int) iter.Seq2[int, error] {
		return Values(slices.Values([]int{n * 10, n * 100}))
	})

	var result []int
	for v := range expanded {
		result = append(result, v)
		if len(result) == 3 {
			break
		}
	}

	assert.Equal(t, []int{10, 100, 20}, result)
}

func TestExpandTypeChange(t *testing.T) {
	// Expand changes types: string → int (simulates device → operations)
	devices := Values(slices.Values([]string{"device-A", "device-B"}))
	ops := Expand(devices, func(name string) iter.Seq2[int, error] {
		// Each "device" has len(name) "operations"
		return Values(slices.Values([]int{len(name)}))
	})

	var result []int
	for v := range ops {
		result = append(result, v)
	}

	assert.Equal(t, []int{8, 8}, result)
}

func TestExpandWithForEach(t *testing.T) {
	// The real pattern: Expand → ForEach
	devices := Values(slices.Values([]string{"dev1", "dev2"}))

	// Each device "has" 2 operations (simulated)
	allOps := Expand(devices, func(device string) iter.Seq2[string, error] {
		return Values(slices.Values([]string{device + "-op1", device + "-op2"}))
	})

	var mu sync.Mutex
	var processed []string
	err := ForEach(context.Background(), allOps, Options{Workers: 1},
		func(_ context.Context, op string) error {
			mu.Lock()
			processed = append(processed, op)
			mu.Unlock()
			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"dev1-op1", "dev1-op2", "dev2-op1", "dev2-op2"}, processed)
}

func TestExpandChained(t *testing.T) {
	// Chain multiple Expand calls: tenants → devices → operations
	tenants := Values(slices.Values([]string{"t1", "t2"}))

	devices := Expand(tenants, func(tenant string) iter.Seq2[string, error] {
		return Values(slices.Values([]string{tenant + "-devA", tenant + "-devB"}))
	})

	ops := Expand(devices, func(device string) iter.Seq2[string, error] {
		return Values(slices.Values([]string{device + "-op1"}))
	})

	var result []string
	for v := range ops {
		result = append(result, v)
	}

	assert.Equal(t, []string{"t1-devA-op1", "t1-devB-op1", "t2-devA-op1", "t2-devB-op1"}, result)
}

// --- Batch tests ---

func TestBatchExactDivisor(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6}))
	var batches [][]int
	for b := range Batch(items, 3) {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]int{{1, 2, 3}, {4, 5, 6}}, batches)
}

func TestBatchRemainder(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))
	var batches [][]int
	for b := range Batch(items, 3) {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]int{{1, 2, 3}, {4, 5}}, batches)
}

func TestBatchSizeOne(t *testing.T) {
	items := Values(slices.Values([]string{"a", "b", "c"}))
	var batches [][]string
	for b := range Batch(items, 1) {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]string{{"a"}, {"b"}, {"c"}}, batches)
}

func TestBatchEmpty(t *testing.T) {
	items := Values(slices.Values([]int{}))
	var batches [][]int
	for b := range Batch(items, 5) {
		batches = append(batches, b)
	}
	assert.Empty(t, batches)
}

func TestBatchSizeLargerThanInput(t *testing.T) {
	items := Values(slices.Values([]int{1, 2}))
	var batches [][]int
	for b := range Batch(items, 100) {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]int{{1, 2}}, batches)
}

func TestBatchZeroSize(t *testing.T) {
	// Zero or negative size should default to 1
	items := Values(slices.Values([]int{1, 2, 3}))
	var batches [][]int
	for b := range Batch(items, 0) {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]int{{1}, {2}, {3}}, batches)
}

func TestBatchEarlyBreak(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
	var batches [][]int
	for b := range Batch(items, 3) {
		batches = append(batches, b)
		if len(batches) == 2 {
			break
		}
	}
	assert.Equal(t, [][]int{{1, 2, 3}, {4, 5, 6}}, batches)
}

func TestBatchWithForEach(t *testing.T) {
	// Real pattern: Batch → ForEach processes each batch
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))
	batched := Batch(items, 2)

	var mu sync.Mutex
	var sums []int
	err := ForEach(context.Background(), batched, Options{},
		func(_ context.Context, batch []int) error {
			sum := 0
			for _, v := range batch {
				sum += v
			}
			mu.Lock()
			sums = append(sums, sum)
			mu.Unlock()
			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []int{3, 7, 5}, sums) // [1+2, 3+4, 5]
}

func TestBatchWithCollect(t *testing.T) {
	items := Values(slices.Values([]int{10, 20, 30, 40}))
	batched := Batch(items, 2)

	results, err := Collect(context.Background(), batched, Options{},
		func(_ context.Context, batch []int) (int, error) {
			sum := 0
			for _, v := range batch {
				sum += v
			}
			return sum, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []int{30, 70}, results)
}

// --- Throttle tests ---

func TestThrottleBasic(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3}))

	start := time.Now()
	var result []int
	for v := range Throttle(items, 50*time.Millisecond) {
		result = append(result, v)
	}
	elapsed := time.Since(start)

	assert.Equal(t, []int{1, 2, 3}, result)
	// 3 items with 50ms gap between items 1→2 and 2→3 = ~100ms minimum
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(90), "should respect throttle interval")
}

func TestThrottleFirstItemImmediate(t *testing.T) {
	items := Values(slices.Values([]int{42}))

	start := time.Now()
	var result []int
	for v := range Throttle(items, 500*time.Millisecond) {
		result = append(result, v)
	}
	elapsed := time.Since(start)

	assert.Equal(t, []int{42}, result)
	assert.Less(t, elapsed.Milliseconds(), int64(100), "first item should be immediate")
}

func TestThrottleEmpty(t *testing.T) {
	items := Values(slices.Values([]int{}))
	var result []int
	for v := range Throttle(items, time.Second) {
		result = append(result, v)
	}
	assert.Empty(t, result)
}

func TestThrottleZeroInterval(t *testing.T) {
	// Zero interval means no throttling — should be fast
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	start := time.Now()
	var result []int
	for v := range Throttle(items, 0) {
		result = append(result, v)
	}
	elapsed := time.Since(start)

	assert.Equal(t, []int{1, 2, 3, 4, 5}, result)
	assert.Less(t, elapsed.Milliseconds(), int64(50), "zero interval should not delay")
}

func TestThrottleEarlyBreak(t *testing.T) {
	items := Values(slices.Values([]int{1, 2, 3, 4, 5}))

	var result []int
	for v := range Throttle(items, 20*time.Millisecond) {
		result = append(result, v)
		if len(result) == 2 {
			break
		}
	}
	assert.Equal(t, []int{1, 2}, result)
}

func TestThrottleWithForEach(t *testing.T) {
	// Real pattern: Throttle → ForEach
	items := Values(slices.Values([]int{1, 2, 3}))
	throttled := Throttle(items, 30*time.Millisecond)

	start := time.Now()
	var mu sync.Mutex
	var processed []int
	err := ForEach(context.Background(), throttled, Options{},
		func(_ context.Context, item int) error {
			mu.Lock()
			processed = append(processed, item)
			mu.Unlock()
			return nil
		},
	)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, processed)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(50), "should respect throttle interval")
}

func TestThrottleAndBatchComposed(t *testing.T) {
	// Compose Throttle + Batch: throttle items, then batch them
	items := Values(slices.Values([]int{1, 2, 3, 4, 5, 6}))
	throttled := Throttle(items, 10*time.Millisecond)
	batched := Batch(throttled, 2)

	var batches [][]int
	for b := range batched {
		batches = append(batches, b)
	}
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5, 6}}, batches)
}
