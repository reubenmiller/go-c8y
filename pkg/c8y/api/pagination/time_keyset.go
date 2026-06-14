package pagination

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// TimeKeysetStrategy walks a time-series collection (events, alarms,
// measurements, operations) by an advancing time-window cursor instead of deep
// offset paging: each page sets dateTo (descending, the default) or dateFrom
// (ascending, revert=true) to the boundary timestamp of the previous page.
//
// Because timestamps are not unique, the boundary is INCLUSIVE so no item is
// ever skipped; the duplicates this re-includes are dropped by a per-boundary
// dedup set carried on the request (skipBoundary/skipIDs). When a single
// timestamp holds more items than a page, the cursor cannot advance, so the
// strategy falls back to offset paging within that timestamp (incrementing
// CurrentPage) until the cluster drains, then resumes the cursor — guaranteeing
// completeness even for pathological clusters.
//
// The cursor seeds (Before/After/Ascending) are set by the entity's ListAll from
// the caller's DateTo/DateFrom/Revert; the entity fetch closure maps them back
// onto its own options.
type TimeKeysetStrategy struct{}

func (TimeKeysetStrategy) Name() string { return "time" }

func (TimeKeysetStrategy) First(base PageRequest) PageRequest {
	base.CurrentPage = 1
	return base
}

// Accept drops items at the previous boundary timestamp that were already
// emitted (the inclusive cursor re-includes them).
func (TimeKeysetStrategy) Accept(req PageRequest, doc jsondoc.JSONDoc) bool {
	if len(req.skipIDs) == 0 {
		return true
	}
	if !docTimestamp(doc).Equal(req.skipBoundary) {
		return true
	}
	_, seen := req.skipIDs[doc.Get("id").String()]
	return !seen
}

func (TimeKeysetStrategy) DefaultReadAhead() int { return 1 }

// Advance moves the cursor to the boundary timestamp of the page (its last item:
// the oldest when descending, the newest when ascending) and records the ids at
// that timestamp for dedup on the next page. If the boundary equals the cursor
// just used, the timestamp spans more than a page, so it offset-pages within the
// window (CurrentPage++) instead of re-issuing the same request.
func (s TimeKeysetStrategy) Advance(prev PageRequest, page PageView) (PageRequest, bool) {
	if page.Count == 0 {
		return prev, false
	}

	// Collect (time,id) for the page; the boundary is the last item's time.
	type ti struct {
		t  time.Time
		id string
	}
	items := make([]ti, 0, page.Count)
	for doc := range page.Docs {
		items = append(items, ti{t: docTimestamp(doc), id: doc.Get("id").String()})
	}
	if len(items) == 0 {
		return prev, false
	}
	boundary := items[len(items)-1].t

	boundaryIDs := make(map[string]struct{})
	for _, it := range items {
		if it.t.Equal(boundary) {
			boundaryIDs[it.id] = struct{}{}
		}
	}

	next := prev

	// Accumulate the dedup set across pages that share the boundary timestamp.
	if boundary.Equal(prev.skipBoundary) {
		for id := range prev.skipIDs {
			boundaryIDs[id] = struct{}{}
		}
	}
	next.skipBoundary = boundary
	next.skipIDs = boundaryIDs

	// Decide cursor advance vs offset fallback within a timestamp cluster.
	cursor := prev.Before
	if prev.Ascending {
		cursor = prev.After
	}
	stuck := !cursor.IsZero() && boundary.Equal(cursor)
	if stuck {
		// The whole window is (at least) one timestamp wider than a page; page
		// through it with offset, keeping the cursor fixed.
		next.CurrentPage = prev.CurrentPage + 1
	} else {
		next.CurrentPage = 1
		if prev.Ascending {
			next.After = boundary
		} else {
			next.Before = boundary
		}
	}

	// A short page is the last page.
	if prev.PageSize > 0 && page.Count < prev.PageSize {
		return next, false
	}
	return next, true
}

// docTimestamp reads an item's logical timestamp, preferring "time" and falling
// back to "creationTime" (operations, audit records). Returns the zero time when
// neither parses.
func docTimestamp(doc jsondoc.JSONDoc) time.Time {
	for _, field := range []string{"time", "creationTime"} {
		if r := doc.Get(field); r.Exists() {
			if t := r.Time(); !t.IsZero() {
				return t
			}
		}
	}
	return time.Time{}
}
