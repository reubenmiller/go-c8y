package pagination

import (
	"fmt"
	"iter"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// StrategyKind selects how a collection is walked. The zero value (Auto) lets
// each entity pick its optimum; callers override it to force a specific
// behaviour. It is a string so it serialises and maps cleanly to a CLI flag.
type StrategyKind string

const (
	// StrategyAuto lets the entity choose the best applicable strategy
	// (id-keyset for inventory, time-keyset for time-series, offset otherwise).
	// The empty string (an unset PaginationOptions.Strategy) is treated
	// identically to "auto", so callers who never set it still get the optimum.
	StrategyAuto StrategyKind = "auto"
	// StrategyOffset is classic currentPage paging.
	StrategyOffset StrategyKind = "offset"
	// StrategyIDKeyset pages inventory by an ascending _id cursor.
	StrategyIDKeyset StrategyKind = "id"
	// StrategyTimeKeyset pages time-series entities by a time-window cursor.
	StrategyTimeKeyset StrategyKind = "time"
)

// PageRequest carries the pagination options for one page plus an optional
// keyset cursor. The per-entity fetch closure applies the cursor fields to its
// own filter options (AfterID -> inventory "_id gt", Before -> dateTo, After ->
// dateFrom). Strategies that don't use a cursor leave those fields zero, so the
// fetch closure behaves exactly like plain offset paging.
type PageRequest struct {
	PaginationOptions

	// AfterID, when set, asks for items whose id is greater than this value
	// (inventory id keyset). Applied as "_id gt 'AfterID'" with "$orderby=_id asc".
	AfterID string

	// Before / After bound a time-series keyset cursor. Before maps to dateTo
	// (descending paging), After maps to dateFrom (ascending paging).
	Before time.Time
	After  time.Time

	// Ascending selects oldest-first paging (Cumulocity revert=true). Default is
	// newest-first.
	Ascending bool

	// skipBoundary/skipIDs carry the time keyset's boundary-dedup state: items at
	// skipBoundary whose id is in skipIDs were already emitted on an earlier page
	// (the inclusive dateTo/dateFrom cursor re-includes the boundary timestamp).
	// They are package-internal cursor state; the entity fetch closure ignores
	// them. Travelling on the request keeps Accept a pure function, so read-ahead
	// cannot corrupt an earlier page's dedup.
	skipBoundary time.Time
	skipIDs      map[string]struct{}
}

// PageView is the read-only view of a freshly fetched page handed to
// Strategy.Advance so it can derive the next cursor.
type PageView struct {
	// Meta is the result metadata (next, totalPages, totalElements, ...).
	Meta map[string]any
	// Count is the number of items the page contained (before Accept filtering).
	Count int
	// Docs iterates the page's raw item documents. Safe to range more than once.
	Docs iter.Seq[jsondoc.JSONDoc]
}

// Strategy drives pagination: it decides the first request, how to advance to
// the next page (or stop), which items to emit (boundary dedup), and the default
// read-ahead depth. A Strategy instance is created per iteration, so it may hold
// per-iteration cursor and dedup state in its fields.
type Strategy interface {
	// Name identifies the strategy (telemetry, debugging).
	Name() string
	// First returns the request for the first page given the resolved base
	// options (page size already normalised).
	First(base PageRequest) PageRequest
	// Advance inspects the page just fetched and returns the next request, or
	// (_, false) to stop. The returned request carries any cursor and
	// per-page dedup state the next page needs, so the strategy stays
	// stateless and read-ahead/parallel safe.
	Advance(prev PageRequest, page PageView) (next PageRequest, more bool)
	// Accept reports whether a page item should be emitted. It is a pure
	// function of the page's own request (which carries the dedup state) and the
	// item, so prefetching a later page can never corrupt an earlier page's
	// filtering. Offset and id keyset return true; time keyset drops boundary
	// duplicates recorded in req.
	Accept(req PageRequest, doc jsondoc.JSONDoc) bool
	// DefaultReadAhead is the prefetch depth used when the caller leaves
	// PaginationOptions.ReadAhead at 0. id keyset returns 0 (exact cursor — no
	// speculative request); time keyset returns 1; offset returns 0 (so the
	// ~40 offset entities keep their exact request pattern unless opted in).
	DefaultReadAhead() int
}

// ParallelStrategy is an optional capability: a strategy whose pages are
// independent once the first page reveals the total. The executor uses Plan to
// fan out the remaining pages concurrently when Concurrency > 1.
type ParallelStrategy interface {
	Strategy
	// Plan returns the independent requests for every page after the bootstrap
	// page, or (_, false) when parallel fan-out does not apply (unknown total,
	// single page, ...).
	Plan(bootstrap PageView, base PageRequest) (reqs []PageRequest, ok bool)
}

// OffsetStrategy is classic Cumulocity currentPage paging: independent pages,
// stop when the server stops returning a next link (or totalPages is reached).
// It is the default for entities without a keyset optimisation and the fallback
// the Paginate wrapper uses.
type OffsetStrategy struct{}

func (OffsetStrategy) Name() string { return "offset" }

// First always starts at page 1, matching the historical ListAll behaviour
// (any CurrentPage in the caller's options is ignored for a full walk).
func (OffsetStrategy) First(base PageRequest) PageRequest {
	base.CurrentPage = 1
	return base
}

func (OffsetStrategy) Advance(prev PageRequest, page PageView) (PageRequest, bool) {
	if page.Count == 0 {
		return prev, false
	}
	if next, ok := page.Meta["next"].(string); !ok || next == "" {
		return prev, false
	}
	if tp, ok := page.Meta["totalPages"].(int64); ok && int64(prev.CurrentPage) >= tp {
		return prev, false
	}
	prev.CurrentPage++
	return prev, true
}

func (OffsetStrategy) Accept(PageRequest, jsondoc.JSONDoc) bool { return true }

func (OffsetStrategy) DefaultReadAhead() int { return 0 }

// Plan enumerates pages 2..totalPages as independent requests for parallel
// fan-out. Returns ok=false when the total is unknown or there is a single page.
func (OffsetStrategy) Plan(bootstrap PageView, base PageRequest) ([]PageRequest, bool) {
	tp, ok := bootstrap.Meta["totalPages"].(int64)
	if !ok || tp <= 1 {
		return nil, false
	}
	reqs := make([]PageRequest, 0, tp-1)
	for p := int64(2); p <= tp; p++ {
		r := base
		r.CurrentPage = int(p)
		reqs = append(reqs, r)
	}
	return reqs, true
}

// ResolveTimeStrategy picks the strategy for a time-series list (events,
// alarms, measurements, operations). Auto and an explicit "time" both select the
// time-window keyset; "offset" selects classic paging; "id" is rejected (the _id
// keyset does not apply to time-series).
func ResolveTimeStrategy(kind StrategyKind) (Strategy, error) {
	switch kind {
	case StrategyOffset:
		return OffsetStrategy{}, nil
	case StrategyTimeKeyset, StrategyAuto, "":
		return TimeKeysetStrategy{}, nil
	case StrategyIDKeyset:
		return nil, fmt.Errorf("pagination strategy %q does not apply to time-series collections; use %q or %q", kind, StrategyTimeKeyset, StrategyOffset)
	default:
		return nil, fmt.Errorf("unknown pagination strategy %q", kind)
	}
}

// resolveReadAhead returns the effective prefetch depth: an explicit
// ReadAhead (>0 depth, <0 off) overrides the strategy's default (ReadAhead==0).
func resolveReadAhead(opts PaginationOptions, s Strategy) int {
	switch {
	case opts.ReadAhead > 0:
		return opts.ReadAhead
	case opts.ReadAhead < 0:
		return 0
	default:
		return s.DefaultReadAhead()
	}
}
