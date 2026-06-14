# Proposal: configurable pagination strategies (keyset optimizations + parallelism)

Status: implemented in the SDK (phases 1–4); CLI: the v2 `devices list` command
migrated (phase 5). Broad CLI rollout follows the per-command v2 migration.
Related: `pkg/c8y/api/pagination` (iterator + options + strategies),
`pkg/c8y/op` (Result), `pkg/c8y/jsondoc`, `docs/OFFLINE_TESTING.md`, go-c8y-cli
`pkg/cmd/devices/list/list.go`, `pkg/request/request.go` (the v1 `_id`
optimization that was ported)

## Implementation status

Shipped in the SDK and verified offline against the fake server:

- `pkg/c8y/api/pagination`: `PageRequest`, `Strategy`/`ParallelStrategy`,
  `StrategyKind`, `OffsetStrategy`, `IDKeysetStrategy`, `TimeKeysetStrategy`,
  `PaginateWith` (with `Paginate` kept as an offset wrapper), and the executor
  (`executor.go`) with read-ahead prefetch + bounded parallel-offset (`rill`).
- `PaginationOptions` gained `Strategy`, `Concurrency`, `ReadAhead` (client-side).
- Inventory family (`managedobjects`, `devices`, `devicegroups`) default to the
  `_id` keyset via `managedobjects.ResolveListStrategy` + `model.WithIDCursor`.
- Time entities (`events`, `alarms`, `measurements`, `operations`) default to the
  time keyset via `pagination.ResolveTimeStrategy`.
- Fake server (`internal/pkg/fakeserver`): `_id`/comparison operators, `$orderby`
  execution, `revert`, deterministic id-tiebreak ordering, precision-adaptive
  date filtering, request capture; plus the invariant harness in
  `test/c8y_api_test/pagination_*_test.go`.

Deviations / discoveries vs. the original proposal:

- **Read-ahead defaults**: id-keyset `0` (off — exact cursor), time-keyset `1`,
  **offset `0`** (off, not `1` — keeps the ~40 offset entities' exact request
  pattern; opt in per call). Fully overridable via `ReadAhead`.
- **Millisecond date cursor (required, not optional)**: the time keyset only
  works if `dateFrom`/`dateTo` are sent at millisecond precision — a
  second-precision cursor truncates sub-second boundaries and loops forever. The
  four time `ListOptions` now carry `layout:"2006-01-02T15:04:05.000Z07:00"` on
  those fields. The code generator (`tools/c8ygen`) must emit this layout for
  date fields so regeneration does not drop it.
- `pkg/c8y/jsondoc/paginator.go` (a second, older, unused paginator) was deleted.

## Goal

Let `*Service.ListAll` choose *how* it walks a collection — classic offset paging,
an `_id` keyset for inventory, or a time-window keyset for the time-series
entities (events/alarms/measurements/operations) — with the optimized strategy
used **by default** and overridable by the caller. The time-window strategy must
not miss entries when timestamps repeat at a page boundary. Everything must be
verifiable offline against the fake server before any live run.

A secondary goal (cheap if the seam is designed in now): allow pages to be
fetched with overlap/concurrency where the strategy permits it.

## Current state

**One chokepoint.** Every `ListAll` (~40) funnels through
`pagination.Paginate[T,D]` (`pkg/c8y/api/pagination/iterator.go:142`). It pages by
incrementing `CurrentPage` and following `Result.Meta["next"]`. The `fetch`
closure it is handed is `func(PaginationOptions) op.Result[D]`, so a strategy can
today only vary page number / page size — it cannot touch the query or the date
window, which is exactly what keyset paging needs.

**Identical per-entity boilerplate** — `managedobjects/service.go:631`,
`events/api.go:70`, `measurements/api.go:70`: copy options, swap in
`PaginationOptions`, call `List`.

**The `_id` optimization already existed in v1 and was dropped in the port.** v1:
`fetchAllInventoryQueryResults` + `optimizeManagedObjectsURL`
(`go-c8y-cli/pkg/request/request.go:519-682`). The v2 device list documents the
regression at `go-c8y-cli/pkg/cmd/devices/list/list.go:194-204` — it appends
`(_id gt '0') $orderby=_id asc` but still pages by `currentPage`: correct, but the
optimization is gone.

**The envelope already carries what a strategy needs.** `op.Result`
(`pkg/c8y/op/result.go:42`) exposes `Meta[next/totalPages/totalElements]`, raw
`Response` bytes, and per-item docs via `D.Iter()`; cursor values read straight
off a doc with `jsondoc.JSONDoc.Get("id"|"time")` (gjson, `jsondoc.go:48`).

**`rill` is already a dependency** (`go.mod`, `github.com/destel/rill`) and is
already used for bounded-concurrency batching in `managedobjects/service.go` — so
bounded, order-preserving fan-out is available without a new dep.

**Offline harness exists** (`docs/OFFLINE_TESTING.md`): `fakeserver.New(t)` +
`testcore.CreateTestClient(t)` / `CreateTestClientWithFakeServer(t)`
(`test/c8y_api_test/testcore/client.go:97,112`), `TEST_MODE=offline|live|record`,
golden tests. **But the fake server cannot exercise either optimization yet** —
see "Fake-server gaps".

## Design

Three additions in `pkg/c8y/api/pagination`, plus a small per-entity cursor
applier. The generic loop moves from "increment a page" to "ask the strategy for
the next request".

### 1. `PageRequest` — `PaginationOptions` plus a generic cursor

```go
type PageRequest struct {
    PaginationOptions
    AfterID   string    // id-keyset: inventory "_id gt 'AfterID'"
    Before    time.Time // time-keyset, descending -> dateTo
    After     time.Time // time-keyset, ascending  -> dateFrom
    Ascending bool
}
```

### 2. `Strategy` — owns the loop (generic, entity-agnostic)

```go
type PageView struct {
    Docs func(yield func(jsondoc.JSONDoc) bool) // page items
    Meta map[string]any                          // next, totalPages, ...
    Count int
}

type Strategy interface {
    Name() string
    First(base PageRequest) PageRequest
    // Advance reads the page just fetched and returns the next request, or
    // (_, false) to stop. Keyset strategies derive the cursor here.
    Advance(prev PageRequest, page PageView) (next PageRequest, more bool)
    // Accept dedups boundary items across pages. Offset/id: always true.
    Accept(doc jsondoc.JSONDoc) bool
    // DefaultReadAhead is the prefetch depth used when the caller leaves
    // ReadAhead at 0. id-keyset and offset return 0; time-keyset returns 1.
    DefaultReadAhead() int
}
```

A `Strategy` is instantiated per `ListAll` call, so it holds the cursor and the
boundary-dedup set as plain fields — no shared state.

### 3. Selection — optimized by default, user-overridable

```go
type StrategyKind string
const (
    StrategyAuto       StrategyKind = "auto"    // entity picks its optimum (DEFAULT); "" is treated the same
    StrategyOffset     StrategyKind = "offset"
    StrategyIDKeyset   StrategyKind = "id"
    StrategyTimeKeyset StrategyKind = "time"
)
// The resolvers treat the empty zero value identically to "auto", so an unset
// PaginationOptions.Strategy still resolves to the optimal strategy. "auto" is a
// real value (not "") so the go-c8y-cli --paginationStrategy flag completes it.

// new fields on PaginationOptions (client-side only; url:"-"):
//   Strategy    StrategyKind  // default Auto
//   Concurrency int           // 0/1 = sequential; >1 = parallel where the strategy allows
//   ReadAhead   int           // 0 = strategy default; <0 = off; >0 = explicit prefetch depth
```

`Auto` resolves per entity: inventory family → id-keyset (but → offset when the
caller supplied a conflicting `$orderby`), the four time entities → time-keyset,
everything else → offset. `Auto` never errors — it picks the best *applicable*
strategy. An **explicit** strategy the request cannot satisfy (e.g. `id` on
measurements, `time` on users, or `id` on inventory with a conflicting
`$orderby`) is a hard error, surfaced via `Iterator.Err()` with no items yielded.

### Backward compatibility

Keep `Paginate(ctx, opts, fetch, constructor)` as a thin wrapper over a new
`PaginateWith(ctx, base, strategy, fetch, constructor)` using `OffsetStrategy`.
The ~40 offset callers are untouched. Only inventory + the time entities adopt the
new form, and each stays ~10 lines — the only entity-specific code is a closure
that maps the cursor onto its own option fields:

```go
// measurements ListAll fetch closure
func(req pagination.PageRequest) op.Result[jsonmodels.Measurement] {
    o := opts; o.PaginationOptions = req.PaginationOptions
    if !req.Before.IsZero() { o.DateTo = req.Before }
    o.Revert = req.Ascending
    return s.List(ctx, o)
}
```

## Correctness — the two keyset algorithms

### ID keyset (inventory)

Order by `_id asc`, filter `_id gt 'lastID'` (start `'0'`), re-request page 1 each
round, advance `lastID` to the last item's id. Because `id` is unique and
monotonic this is exact — no skips, no duplicates. The parenthesization logic from
v1's `optimizeManagedObjectsURL` becomes a `model.InventoryQuery` helper
(`WithIDCursor`). Only applied by `Auto` when the caller has not requested a
different `$orderby` (otherwise their sort would be silently overridden → fall
back to offset).

### Time keyset (events / alarms / measurements / operations)

The subtle case: timestamps are not unique, so a naive cursor skips or duplicates
items at a page boundary.

1. Descending (C8Y default): advance `dateTo = lastTime` **inclusive** each page.
2. **Boundary dedup:** remember the ids already emitted *at* `lastTime`; skip them
   when they reappear on the next page. Inclusive boundary ⇒ never skip; dedup ⇒
   never duplicate. Reset the set when the boundary time advances.
3. **Cluster fallback:** if an entire page is a single timestamp equal to the
   boundary (a timestamp cluster larger than `pageSize`), `dateTo` cannot advance.
   Detect it (new boundary == previous `dateTo` and no new items accepted) and
   offset-page *within* that timestamp until drained, then resume keyset. This is
   the completeness backstop.
4. Ascending mirror: advance `dateFrom = lastTime` with `revert=true`.

```
items newest -> oldest, pageSize 3:   A(10:03) B(10:02) C(10:01) D(10:01) E(10:00)
page 1 fetched: A,B,C   boundary = 10:01 (C), but D is also 10:01 and unfetched
  naive  dateTo < 10:01 : page 2 = {E}            -> D skipped            (WRONG)
  keyset dateTo <= 10:01, skip {C}: page 2 = {C,D,E} -> drop C, emit D,E  (COMPLETE)
```

## Parallelism

Keyset is **sequential by construction** — page N+1's request is derived from page
N's data, so its pages cannot be issued ahead of time. Offset is the opposite:
once the first page reveals `totalPages`, every page URL is independent and
parallelizable. The two goals partly compete (keyset minimizes server-side work
and round-trips; parallel-offset minimizes wall-clock at the cost of many
concurrent, server-heavy deep-offset queries), so parallelism is opt-in and lives
in the executor, not in any entity.

### Seam: an optional capability interface

```go
type ParallelStrategy interface {
    Strategy
    // After the bootstrap page reveals totalPages, return the remaining
    // independent requests. Offset implements this; keyset returns ok=false.
    Plan(bootstrap PageView, base PageRequest) (reqs []PageRequest, ok bool)
}
```

`PaginateWith` fetches page 1, then: if the strategy is a `ParallelStrategy`,
`Concurrency > 1`, and `Plan` returns `ok`, fan pages `2..N` through a bounded
`rill` pool (order-preserving by default; opt-in as-completed). Otherwise run the
sequential `Advance` loop. No entity code changes.

### The wins, by effort

| Approach | Works with | Win | Cost / risk | Effort |
| --- | --- | --- | --- | --- |
| 1-page read-ahead (prefetch next while consumer drains current) | all strategies, incl. keyset | overlaps next fetch with client-side processing | minimal | easy — universal |
| Bounded parallel offset (fan out once totalPages known) | offset only | high on stable/bounded sets | server-heavy; 429s; offset drift on live data | easy-ish, gated behind `Concurrency` |
| Partitioned keyset (shard id-range or time-window, keyset per shard) | keyset | high | uneven shards; boundary correctness | complex — later |

Read-ahead overlaps the next fetch with downstream work: even for keyset, page
N+1's cursor is known the instant page N's bytes arrive, so a depth-1 prefetch can
run while the consumer drains page N. It is a bounded buffer in the sequential
executor, **controlled per strategy**, because a speculative prefetch can cost one
extra request when the consumer stops early or a full last page is followed by an
empty one. Defaults (as shipped): **off for id-keyset** (the cursor is exact and
the whole point is to stay gentle on the server), **off for offset** (preserves
the existing entities' exact request pattern), **on (depth 1) for time-keyset**.
The caller overrides per call via `ReadAhead` (`<0` forces off, `>0` sets the
depth), so the defaults are only a starting point to tune.

### Caveats baked into the executor

- Deep offset paging is expensive in Cumulocity; bound `Concurrency` and honor
  `Retry-After`/429 (`op.Result` already models retryability).
- `totalPages`/`totalElements` can be costly or stale on large/live collections;
  parallel offset is *more* exposed to skip/duplicate-on-mutation than serial —
  the very failure keyset avoids. Document the trade-off at the call site.
- With `MaxItems` set, cap the fan-out to `ceil(MaxItems/pageSize)` pages.
- Default to order-preserving emission to keep `Items()` semantics.

## Fake-server gaps to close first

These additions are also how we *verify* the strategies offline
(`internal/pkg/fakeserver`):

- Inventory `$filter` comparison operators `gt`/`lt`/`ge`/`le` and real
  `$orderby=_id asc|desc` execution — today `handler_inventory.go` strips
  `$orderby` and `applyCQLFilter` only does `eq`/`has()`.
- `revert=true|false` for time entities — today reverse order is hard-coded
  (`filtering.go` `ReverseItems`).
- Compare `dateFrom`/`dateTo` at **millisecond** precision — today truncated to
  seconds (`filtering.go:130,146`), which masks the exact border case.
- Stable id tiebreak when timestamps are equal — today reverse-insertion order,
  non-deterministic vs real C8Y.
- Seed adversarial datasets: duplicate-timestamp clusters, including one larger
  than `pageSize`, plus controlled id ranges.

## Testing approach

1. **Strategy unit tests** — feed canned page envelopes; assert cursor math,
   dedup set, cluster fallback, stop conditions. No server.
2. **Integration over the fake server** — seeded adversarial data through real
   `ListAll`.
3. **Invariants (the core assertions, table/property-based):** for each dataset ×
   strategy, the concatenation of pages equals the offset baseline **as a set**
   (no skips), with **zero duplicates**, and respects `MaxItems`. Run offset vs
   keyset over identical data and assert equal sets.
4. **Request-count assertion** via the recorder — confirm keyset issues fewer/
   lighter requests, and parallel-offset overlaps them.
5. **Golden + record→live** — `TEST_MODE=record` against a tenant, replay, then
   `TEST_MODE=live` as the final gated step.

## CLI adoption (go-c8y-cli)

- List commands move to `Auto`; drop the manual `_id gt '0'` hack in
  `pkg/cmd/devices/list/list.go`. Inventory family gets keyset for free via
  `c8ystream.ListCall` (`pkg/c8ystream/list.go:25`), which already drives
  `ListAll` — no per-command change beyond removing the hack.
- Time-based list commands get the time strategy under `--includeAll`.
- Expose `--pagination-strategy auto|offset|id|time` (+ config default) and,
  optionally, `--pagination-concurrency`. `--includeAll` / `--maxItems` semantics
  unchanged.

## Suggested path (incremental, each step independently useful)

1. **Fake-server gaps + invariant harness.** Close the gaps above and add the
   set-equality / no-dup / no-skip harness against the *existing* offset paths.
   Proves we can detect skips/dups before any strategy is written.
2. **SDK core.** `PageRequest`, `Strategy`, `PaginateWith`, `OffsetStrategy`
   (incl. `ParallelStrategy.Plan`), the `ReadAhead` prefetch, and `Concurrency`
   wiring; keep `Paginate` as a wrapper. No behavior change; full suite green;
   read-ahead default resolved per strategy (id off, time/offset on at depth 1).
3. **ID keyset** + inventory family wiring + `model.WithIDCursor` + tests.
4. **Time keyset** (dedup + cluster fallback) + the four time entities +
   adversarial tests.
5. **CLI** — `Auto` by default, strategy/concurrency flags, remove the v1 hack.
6. **Docs + cleanup** — finalize this doc, changelog, and reconcile/remove the
   second paginator in `pkg/c8y/jsondoc/paginator.go` if unused.

## Effort and risk

- Step 1 is the gate and is self-contained (fake-server + tests only).
- Step 2 is mechanical and proven by the unchanged suite; read-ahead and the
  parallel seam land here so later strategies inherit them.
- Step 3 is low-risk (unique key, exact cursor).
- Step 4 carries the real correctness risk (boundary dedup, >pageSize clusters) —
  mitigated entirely by the Step 1 invariant harness with adversarial seeds.
- Parallel-offset is opt-in (`Concurrency` defaults to sequential), so it cannot
  regress default behavior or add server load unless explicitly requested.

## Open decisions (defaulted; flip any)

- Config surface: enum field on `PaginationOptions` (serializable, CLI-flag-
  friendly) over a functional option. **Recommended.**
- Inapplicable explicit strategy: hard error via `Iterator.Err()` (Auto never
  errors — it falls back to the best applicable strategy).
- Scope: inventory family + the four named time entities; others stay offset.
- `ReadAhead`: strategy-specific default (id-keyset off; offset off; time-keyset
  on at depth 1), overridable per call; `Concurrency` default sequential (off).
