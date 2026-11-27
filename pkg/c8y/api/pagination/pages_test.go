package pagination

import (
	"context"
	"fmt"
	"iter"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
)

// fakeCollection is a minimal JSONDocument: its Iter walks the per-item docs,
// mirroring how jsonmodels collection types expose the plucked array.
type fakeCollection struct {
	items []string
}

func (f fakeCollection) Iter() iter.Seq[jsondoc.JSONDoc] {
	return func(yield func(jsondoc.JSONDoc) bool) {
		for _, raw := range f.items {
			if !yield(jsondoc.New([]byte(raw))) {
				return
			}
		}
	}
}

// twoPageFetch returns a fetch closure for a 2-page collection (2 items then 1
// item). Each page's Response carries the full envelope (array + statistics +
// paging links), exactly as ExecuteCollection now retains it.
func twoPageFetch() func(PaginationOptions) op.Result[fakeCollection] {
	page1 := op.WithMeta(op.WithMeta(op.WithMeta(op.WithResponse(op.OK(fakeCollection{items: []string{`{"id":"1"}`, `{"id":"2"}`}}), []byte(`{"managedObjects":[{"id":"1"},{"id":"2"}],"next":"https://x/page2","statistics":{"currentPage":1,"pageSize":2,"totalPages":2,"totalElements":3}}`)), "next", "https://x/page2"), "totalPages", int64(2)), "totalElements", int64(3))
	page2 := op.WithMeta(op.WithMeta(op.WithMeta(op.WithResponse(op.OK(fakeCollection{items: []string{`{"id":"3"}`}}), []byte(`{"managedObjects":[{"id":"3"}],"statistics":{"currentPage":2,"pageSize":2,"totalPages":2,"totalElements":3}}`)), "next", ""), "totalPages", int64(2)), "totalElements", int64(3))

	return func(opts PaginationOptions) op.Result[fakeCollection] {
		switch max(opts.CurrentPage, 1) { // an unset currentPage is the first page
		case 1:
			return page1
		case 2:
			return page2
		default:
			return op.OK(fakeCollection{}) // empty page beyond the end
		}
	}
}

func collectPages(t *testing.T, it *Iterator[string]) []jsondoc.JSONDoc {
	t.Helper()
	var docs []jsondoc.JSONDoc
	for doc, err := range it.Pages() {
		assert.NoError(t, err)
		docs = append(docs, doc)
	}
	return docs
}

// Unbounded pagination yields one envelope per page, each carrying the whole
// un-plucked document (array + statistics), not just the items.
func TestPages_Unbounded(t *testing.T) {
	it := Paginate(context.Background(), PaginationOptions{}, twoPageFetch(),
		func(b []byte) string { return string(b) })

	docs := collectPages(t, it)

	assert.Len(t, docs, 2, "should yield one envelope per page")
	// First page is the full envelope, not the plucked array.
	assert.True(t, docs[0].Get("managedObjects").IsArray())
	assert.Equal(t, int64(3), docs[0].Get("statistics.totalElements").Int())
	assert.Equal(t, "https://x/page2", docs[0].Get("next").String())
	assert.Equal(t, "3", docs[1].Get("managedObjects.0.id").String())
}

// A capped query (the plain, non-includeAll case: MaxItems == one page) stops
// after the first page — matching the historical single-page raw behaviour.
func TestPages_SinglePageWhenCapped(t *testing.T) {
	it := Paginate(context.Background(), PaginationOptions{MaxItems: 2}, twoPageFetch(),
		func(b []byte) string { return string(b) })

	docs := collectPages(t, it)

	assert.Len(t, docs, 1, "MaxItems reached on page 1 should stop pagination")
	assert.Equal(t, int64(2), docs[0].Get("managedObjects.#").Int())
}

// Pages() and Items() are independent views over the same query.
func TestPages_ItemsConsistency(t *testing.T) {
	it := Paginate(context.Background(), PaginationOptions{}, twoPageFetch(),
		func(b []byte) string { return string(b) })

	var items []string
	for item, err := range it.Items() {
		assert.NoError(t, err)
		items = append(items, item)
	}
	assert.Equal(t, []string{`{"id":"1"}`, `{"id":"2"}`, `{"id":"3"}`}, items)
}

// NewIterator has no underlying page envelopes, so Pages() is an empty seq.
func TestPages_NewIteratorEmpty(t *testing.T) {
	it := NewIterator(func(yield func(string, error) bool) {
		yield("a", nil)
	})
	assert.Empty(t, collectPages(t, it))
}

// SinglePage sends the caller's request unchanged (CurrentPage kept, no
// forced currentPage=1, no keyset cursor) and fetches exactly one page even
// when the server reports a next page.
func TestSinglePage_SendsRequestUnchanged(t *testing.T) {
	var seen []PageRequest
	it := PaginateWith(
		context.Background(),
		PageRequest{PaginationOptions: PaginationOptions{PageSize: 2, CurrentPage: 0, SinglePage: true}},
		IDKeysetStrategy{},
		func(req PageRequest) op.Result[fakeCollection] {
			seen = append(seen, req)
			return twoPageFetch()(PaginationOptions{CurrentPage: 1})
		},
		func(b []byte) string { return string(b) },
	)
	var items []string
	for item, err := range it.Items() {
		assert.NoError(t, err)
		items = append(items, item)
	}
	assert.Len(t, items, 2)
	if assert.Len(t, seen, 1) {
		assert.Equal(t, 0, seen[0].CurrentPage, "the caller's currentPage is not forced to 1")
		assert.Empty(t, seen[0].AfterID, "no keyset cursor is added")
	}
}

// The caller's explicit page is kept.
func TestSinglePage_KeepsCurrentPage(t *testing.T) {
	var seen []PageRequest
	it := PaginateWith(
		context.Background(),
		PageRequest{PaginationOptions: PaginationOptions{PageSize: 2, CurrentPage: 3, SinglePage: true}},
		OffsetStrategy{},
		func(req PageRequest) op.Result[fakeCollection] {
			seen = append(seen, req)
			return twoPageFetch()(PaginationOptions{CurrentPage: 1})
		},
		func(b []byte) string { return string(b) },
	)
	for range it.Items() {
	}
	if assert.Len(t, seen, 1) {
		assert.Equal(t, 3, seen[0].CurrentPage)
	}
}

// The walk of the offset strategy sends no currentPage on its first request and
// no totals on the next ones (only the caller's flags), like the plain list
// request: currentPage 2 follows.
func TestOffsetWalk_RequestsOfTheCaller(t *testing.T) {
	var seen []PaginationOptions
	fetch := twoPageFetch()
	it := Paginate(context.Background(), PaginationOptions{PageSize: 2}, func(opts PaginationOptions) op.Result[fakeCollection] {
		seen = append(seen, opts)
		return fetch(opts)
	}, func(b []byte) string { return string(b) })

	for _, err := range it.Items() {
		assert.NoError(t, err)
	}

	assert.Len(t, seen, 2)
	assert.Equal(t, 0, seen[0].CurrentPage, "the first request has no currentPage")
	assert.Equal(t, 2, seen[1].CurrentPage)
	for _, opts := range seen {
		assert.False(t, opts.WithTotalPages, "totals are not forced")
		assert.False(t, opts.WithTotalElements, "totals are not forced")
	}
}

// timeSeriesFetch serves n items (id n is at the second n, newest first) like a
// time-series collection: dateTo (the cursor Before, inclusive) and currentPage
// select the page.
func timeSeriesFetch(n int, requests *atomic.Int32) func(PageRequest) op.Result[fakeCollection] {
	return func(req PageRequest) op.Result[fakeCollection] {
		requests.Add(1)
		var items []string
		for id := n; id >= 1; id-- {
			ts := time.Date(2024, 1, 1, 10, 0, id, 0, time.UTC)
			if !req.Before.IsZero() && ts.After(req.Before) {
				continue
			}
			items = append(items, fmt.Sprintf(`{"id":"%d","time":"%s"}`, id, ts.Format("2006-01-02T15:04:05.000Z")))
		}
		page := max(req.CurrentPage, 1)
		start := min((page-1)*req.PageSize, len(items))
		end := min(start+req.PageSize, len(items))
		return op.OK(fakeCollection{items: items[start:end]})
	}
}

// MaxItems items are returned by the time keyset although every page after the
// first starts with the boundary item of the previous page (a duplicate which is
// dropped), and the walk stops when they are read.
func TestTimeKeyset_MaxItemsIsReached(t *testing.T) {
	var requests atomic.Int32
	it := PaginateWith(context.Background(),
		PageRequest{PaginationOptions: PaginationOptions{PageSize: 2, MaxItems: 6}},
		TimeKeysetStrategy{}, timeSeriesFetch(10, &requests),
		func(b []byte) string { return string(jsondoc.New(b).Get("id").String()) })

	var ids []string
	for id, err := range it.Items() {
		assert.NoError(t, err)
		ids = append(ids, id)
	}

	assert.Equal(t, []string{"10", "9", "8", "7", "6", "5"}, ids)
	// 5 pages hold the 6 items; the read-ahead may have fetched up to 2 more
	assert.LessOrEqual(t, requests.Load(), int32(7), "the walk stops once MaxItems items are read")
}

// One page of items (MaxItems == page size) is one request.
func TestTimeKeyset_OnePageIsOneRequest(t *testing.T) {
	var requests atomic.Int32
	it := PaginateWith(context.Background(),
		PageRequest{PaginationOptions: PaginationOptions{PageSize: 2, MaxItems: 2}},
		TimeKeysetStrategy{}, timeSeriesFetch(10, &requests),
		func(b []byte) string { return string(jsondoc.New(b).Get("id").String()) })

	var ids []string
	for id, err := range it.Items() {
		assert.NoError(t, err)
		ids = append(ids, id)
	}

	assert.Equal(t, []string{"10", "9"}, ids)
	assert.Equal(t, int32(1), requests.Load())
}
