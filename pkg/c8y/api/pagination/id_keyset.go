package pagination

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// IDKeysetStrategy walks an inventory collection by an ascending _id cursor: it
// always re-requests "page 1" with the filter "_id gt '<last id>'" ordered by
// "_id asc", so each request is an index seek rather than a deep offset scan and
// the walk stops as soon as a short page arrives. Because id is unique and
// monotonic the enumeration is exact — no skips, no duplicates — so Accept is a
// no-op and read-ahead defaults off (the cursor is exact; a speculative request
// would only waste a round-trip).
//
// The cursor is carried in PageRequest.AfterID; the entity's fetch closure
// translates it into its own query field via model.WithIDCursor.
type IDKeysetStrategy struct{}

func (IDKeysetStrategy) Name() string { return "id" }

// First seeds the cursor at "0" (every real id is greater) so the first page is
// also served by the keyset filter.
func (IDKeysetStrategy) First(base PageRequest) PageRequest {
	base.CurrentPage = 1
	if base.AfterID == "" {
		base.AfterID = "0"
	}
	return base
}

// Advance moves the cursor to the largest id on the page (the last item, since
// the page is ordered _id asc). A page shorter than the page size is the last
// page, which lets the walk stop without the extra empty request offset paging
// would need.
func (IDKeysetStrategy) Advance(prev PageRequest, page PageView) (PageRequest, bool) {
	if page.Count == 0 {
		return prev, false
	}
	lastID := ""
	for doc := range page.Docs {
		lastID = doc.Get("id").String()
	}
	if lastID == "" {
		return prev, false
	}
	next := prev
	next.CurrentPage = 1
	next.AfterID = lastID
	if prev.PageSize > 0 && page.Count < prev.PageSize {
		return next, false
	}
	return next, true
}

func (IDKeysetStrategy) Accept(PageRequest, jsondoc.JSONDoc) bool { return true }

func (IDKeysetStrategy) DefaultReadAhead() int { return 0 }
