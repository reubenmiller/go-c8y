# Proposal: generating go-c8y-cli commands and go-c8y v2 services from one spec

Status: assessment / not started
Related: `pkg/c8y/output` (streaming output pipeline), go-c8y-cli `feat/v2-output-streaming`

## Current state

**go-c8y-cli** is already fully spec-driven: 65 JSON specs under `api/spec/json/`
(YAML sources under `api/spec/yaml/`) are turned into 366 `*.auto.go` cobra
commands (~200 lines each) by a PowerShell templating engine
(`scripts/build-cli/New-C8yApiGoCommand.ps1`, ~800 lines). The spec is rich and
battle-tested — beyond method/path/params it encodes CLI-specific UX knowledge:

- flag names, types, and Cumulocity query-language `format` strings
  (e.g. `(name eq '%s')`)
- pipeline support and `pipelineAliases` for stdin streaming
- completion sources (device/user/application lookups)
- per-language aliases (go, powershell) and tested usage examples

**go-c8y v2** services (operations, inventory, alarms, ...) are hand-written:
typed `Options` structs, `Result[T]`, `pagination.Iterator[T]`.

## Options considered

**A. Spec-first: one spec, two generated backends (recommended).**
Treat the CLI's existing spec as the single source of truth and write one
Go-based generator (`text/template` + `go:generate`) with two emitters:

1. *v2 services*: `queryParameters` → `ListOptions` struct fields,
   `pathParameters`/`body` → typed create/update options, `collectionProperty`
   → `Paginate` wiring. The mapping is almost mechanical.
2. *CLI commands*: thin cobra wrappers that parse flags, call the generated v2
   service, and hand the iterator to the `output` package for
   filter/select/template/render. A generated command shrinks from ~200 lines
   of request assembly to ~40 lines of flag-to-options mapping, because
   pagination, output shaping and views live in the SDK.

**B. Generate the CLI from v2 source code (reflection/AST over Options
structs).** Rejected: the SDK code cannot express the CLI-only knowledge
(pipeline aliases, completions, query formats, examples). It would have to be
re-attached via struct tags or sidecar files — a worse, fragmented spec.

**C. Generate from Cumulocity's OpenAPI specs.** Rejected as primary source
for the same reason — the UX metadata doesn't exist there. (OpenAPI can still
be used to *diff* the spec for missing endpoints/parameters.)

## Suggested path (incremental, each step independently useful)

1. **Port the generator to Go**, consuming the existing JSON specs unchanged
   and reproducing today's `*.auto.go` byte-for-byte (golden tests make this
   verifiable). Removes the PowerShell build dependency; no behavior change.
2. **Add the v2-service emitter.** Start with one group (e.g. operations) and
   compare against the hand-written service to converge naming idioms. The
   spec moves to (or is vendored by) the go-c8y repo so SDK and CLI generate
   from the same revision.
3. **Add the thin-command emitter** targeting v2 + `output`. Migrate
   group-by-group with both command styles coexisting behind the same root
   command — the same fallback pattern used for the jsonfilter engine swap.
4. **Runtime helpers stay as libraries.** The CLI's `pkg/flags`,
   `c8yfetcher` lookups and completion helpers remain hand-written runtime
   code; the generator only wires names into them, as it does today.

## Effort and risk

- Step 1 is bounded: one 800-line template script to port, with byte-exact
  output as the acceptance test.
- Step 2 risk is naming/idiom churn in v2 — mitigated by generating into a
  separate package until reviewed.
- Step 3 carries the long-tail risk (366 commands, many edge behaviors);
  the group-by-group migration plus the commander test suite (which asserts
  on dry-run request construction, unaffected by the output layer) keeps each
  step verifiable.
