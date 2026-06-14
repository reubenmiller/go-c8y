package pagination

import (
	"context"
	"iter"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// mockColl is a minimal JSONDocument collection of pre-built item docs.
type mockColl struct {
	items []jsondoc.JSONDoc
}

func (m mockColl) Iter() iter.Seq[jsondoc.JSONDoc] {
	return func(yield func(jsondoc.JSONDoc) bool) {
		for _, d := range m.items {
			if !yield(d) {
				return
			}
		}
	}
}

// mockServer serves a fixed set of sequential ids (1..total) as offset pages.
type mockServer struct {
	total     int
	calls     atomic.Int64
	inflight  atomic.Int64
	maxInPlay atomic.Int64
	delay     time.Duration
}

func (s *mockServer) fetch(req PageRequest) op.Result[mockColl] {
	s.calls.Add(1)
	n := s.inflight.Add(1)
	for {
		m := s.maxInPlay.Load()
		if n <= m || s.maxInPlay.CompareAndSwap(m, n) {
			break
		}
	}
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.inflight.Add(-1)

	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 1
	}
	page := req.CurrentPage
	if page < 1 {
		page = 1
	}
	totalPages := (s.total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > s.total {
		start = s.total
	}
	if end > s.total {
		end = s.total
	}
	items := make([]jsondoc.JSONDoc, 0, end-start)
	for i := start; i < end; i++ {
		items = append(items, jsondoc.New([]byte(`{"id":"`+strconv.Itoa(i+1)+`"}`)))
	}
	meta := map[string]any{
		"totalPages":    int64(totalPages),
		"totalElements": int64(s.total),
		"currentPage":   int64(page),
	}
	if page < totalPages {
		meta["next"] = "https://example/next"
	}
	return op.Result[mockColl]{Data: mockColl{items: items}, Meta: meta, Response: []byte("{}")}
}

func idCtor(b []byte) string { return jsondoc.New(b).Get("id").String() }

func collect(it *Iterator[string]) ([]string, error) {
	var got []string
	for v, err := range it.Items() {
		if err != nil {
			return got, err
		}
		got = append(got, v)
	}
	return got, it.Err()
}

func wantIDs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = strconv.Itoa(i + 1)
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Every mode must enumerate the full set in the same order.
func TestExecutor_AllModesPreserveOrderAndCompleteness(t *testing.T) {
	const total = 237
	want := wantIDs(total)
	modes := map[string]PaginationOptions{
		"sequential": {PageSize: 10},
		"read-ahead": {PageSize: 10, ReadAhead: 4},
		"parallel":   {PageSize: 10, Concurrency: 5},
	}
	for name, opts := range modes {
		srv := &mockServer{total: total}
		it := PaginateWith(context.Background(), PageRequest{PaginationOptions: opts}, OffsetStrategy{}, srv.fetch, idCtor)
		got, err := collect(it)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !eq(got, want) {
			t.Fatalf("%s: got %d ids, want %d (order/completeness mismatch)", name, len(got), len(want))
		}
	}
}

// MaxItems must cap every mode to exactly the first N ids, in order.
func TestExecutor_MaxItemsAllModes(t *testing.T) {
	const total = 250
	want := wantIDs(total)[:55]
	for _, opts := range []PaginationOptions{
		{PageSize: 10, MaxItems: 55},
		{PageSize: 10, MaxItems: 55, ReadAhead: 4},
		{PageSize: 10, MaxItems: 55, Concurrency: 5},
	} {
		srv := &mockServer{total: total}
		it := PaginateWith(context.Background(), PageRequest{PaginationOptions: opts}, OffsetStrategy{}, srv.fetch, idCtor)
		got, err := collect(it)
		if err != nil {
			t.Fatal(err)
		}
		if !eq(got, want) {
			t.Fatalf("MaxItems opts=%+v: got %d, want 55", opts, len(got))
		}
	}
}

// Parallel mode must actually overlap requests; sequential and read-ahead must
// keep a single request in flight (one producer goroutine).
func TestExecutor_ConcurrencyInFlight(t *testing.T) {
	run := func(opts PaginationOptions) int64 {
		srv := &mockServer{total: 200, delay: 15 * time.Millisecond}
		it := PaginateWith(context.Background(), PageRequest{PaginationOptions: opts}, OffsetStrategy{}, srv.fetch, idCtor)
		_, err := collect(it)
		if err != nil {
			t.Fatal(err)
		}
		return srv.maxInPlay.Load()
	}
	if m := run(PaginationOptions{PageSize: 10}); m != 1 {
		t.Errorf("sequential maxInFlight = %d, want 1", m)
	}
	if m := run(PaginationOptions{PageSize: 10, ReadAhead: 4}); m != 1 {
		t.Errorf("read-ahead maxInFlight = %d, want 1 (single producer)", m)
	}
	if m := run(PaginationOptions{PageSize: 10, Concurrency: 5}); m < 2 {
		t.Errorf("parallel maxInFlight = %d, want >= 2", m)
	}
}

// Stopping the consumer early must not fetch the whole collection: read-ahead is
// bounded by its depth, parallel by its concurrency.
func TestExecutor_EarlyStopBoundsFetches(t *testing.T) {
	check := func(opts PaginationOptions, maxCalls int64) {
		srv := &mockServer{total: 1000} // 100 pages at pageSize 10
		it := PaginateWith(context.Background(), PageRequest{PaginationOptions: opts}, OffsetStrategy{}, srv.fetch, idCtor)
		n := 0
		for _, err := range it.Items() {
			if err != nil {
				t.Fatal(err)
			}
			n++
			if n == 5 { // stop within the first page
				break
			}
		}
		// Allow producers/workers to settle.
		time.Sleep(30 * time.Millisecond)
		if got := srv.calls.Load(); got > maxCalls {
			t.Errorf("opts=%+v: %d fetches after early stop, want <= %d", opts, got, maxCalls)
		}
	}
	check(PaginationOptions{PageSize: 10}, 1)                 // sequential: only page 1
	check(PaginationOptions{PageSize: 10, ReadAhead: 4}, 6)   // page 1 + up to depth prefetch
	check(PaginationOptions{PageSize: 10, Concurrency: 5}, 6) // page 1 + up to concurrency
}

// A terminal (empty) first page must not trigger any speculative prefetch — the
// dry-run guarantee.
func TestExecutor_EmptyFirstPageNoPrefetch(t *testing.T) {
	for _, opts := range []PaginationOptions{
		{PageSize: 10, ReadAhead: 4},
		{PageSize: 10, Concurrency: 5},
	} {
		srv := &mockServer{total: 0}
		it := PaginateWith(context.Background(), PageRequest{PaginationOptions: opts}, OffsetStrategy{}, srv.fetch, idCtor)
		got, err := collect(it)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("empty collection yielded %d items", len(got))
		}
		if c := srv.calls.Load(); c != 1 {
			t.Errorf("opts=%+v: %d fetches for empty collection, want 1", opts, c)
		}
	}
}
