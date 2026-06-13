package pagination

import (
	"context"
	"iter"
	"testing"

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
	page1 := op.OK(fakeCollection{items: []string{`{"id":"1"}`, `{"id":"2"}`}}).
		WithResponse([]byte(`{"managedObjects":[{"id":"1"},{"id":"2"}],"next":"https://x/page2","statistics":{"currentPage":1,"pageSize":2,"totalPages":2,"totalElements":3}}`)).
		WithMeta("next", "https://x/page2").
		WithMeta("totalPages", int64(2)).
		WithMeta("totalElements", int64(3))
	page2 := op.OK(fakeCollection{items: []string{`{"id":"3"}`}}).
		WithResponse([]byte(`{"managedObjects":[{"id":"3"}],"statistics":{"currentPage":2,"pageSize":2,"totalPages":2,"totalElements":3}}`)).
		WithMeta("next", "").
		WithMeta("totalPages", int64(2)).
		WithMeta("totalElements", int64(3))

	return func(opts PaginationOptions) op.Result[fakeCollection] {
		switch opts.CurrentPage {
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
