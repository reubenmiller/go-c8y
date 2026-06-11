# output — streaming response processing (proof of concept)

A lazy, pull-based pipeline for filtering, shaping and rendering Cumulocity
collection responses, intended to replace the buffered per-row re-parsing
pipeline in go-c8y-cli.

## Design

- **Pull-based on `iter.Seq2`** — nothing executes until the renderer asks for
  the next item; stopping consumption stops the source (including pagination).
  Laziness and backpressure with no channels or goroutines.
- **Compile once, run per row** — filter globs/regexes and select patterns are
  compiled when a stage is constructed; per item the cost is gjson path
  lookups only. No flatten/sort/unflatten cycles, no query-engine re-parsing.
- **Client-agnostic sources** — `FromIterator` (v2 `pagination.Iterator`),
  `FromBytes` (an already-buffered body, zero-copy slicing), and `FromReader`
  (incremental decode straight off an HTTP response body). go-c8y-cli can
  adopt this layer via `FromBytes(resp.Body(), "managedObjects")` without
  regenerating commands or migrating to the v2 client.
- **Custom logic is a function** — a `Stage` is
  `func(iter.Seq2[jsondoc.JSONDoc, error]) iter.Seq2[jsondoc.JSONDoc, error]`;
  `Map`/`Filter`/`Head` cover the common cases.

```go
err := output.Render(ctx,
    output.FromBytes(body, "managedObjects"),
    encode.NewCSV(os.Stdout, encode.CSVOptions{Header: true,
        Columns: []string{"id", "name", "c8y_Hardware.serialNumber"}}),
    output.Filter(filter.Like("name", "linux*")),
    shape.Select("id", "name", "c8y_Hardware.*"),
)
```

## Benchmark results

Fixture mirrors the motivating case: 2000 managed objects, ~33MB body, which
takes ~3s to process in go-c8y-cli. Apple M1 Max:

```
go test -bench . -benchmem -run '^$' ./pkg/c8y/output/

BenchmarkPassthroughNDJSON         31.3ms   1114 MB/s      66KB allocs
BenchmarkFilterNDJSON              31.7ms   1101 MB/s     114KB allocs
BenchmarkFilterSelectCSV           34.8ms   1003 MB/s      25MB allocs
BenchmarkSelectJSONArray           71.5ms    488 MB/s      41MB allocs
BenchmarkStreamingReaderNDJSON    202.1ms    173 MB/s      37MB allocs
BenchmarkHead10FromReader           1.0ms   (early exit on 33MB body)
```

~3s in go-c8y-cli → 31–72ms here (40–95x), with peak memory bounded by one
item plus I/O buffers in the streaming path. `Head10FromReader` demonstrates
laziness end to end: taking 10 items from a 33MB stream costs ~1ms because
the rest is never read.

## PoC shortcuts / next steps

- `FromReader` uses `encoding/json`'s tokenizer (~170 MB/s). A dedicated
  depth-counting scanner should reach gjson-like speeds if the streaming path
  becomes the default; it already overlaps processing with network transfer,
  which dominates wall clock.
- `shape.Select` builds output via per-match `sjson` sets; a single-walk
  streaming writer would cut the wildcard-select allocations further.
- Not yet implemented from the proposal: table renderer with sampled column
  widths, view definitions, jsonnet/gojq template stages, a parser for the
  go-c8y-cli filter language (the predicate building blocks exist), and an
  `ExecuteStream` variant on the core client to feed `FromReader` directly.
