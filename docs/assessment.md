# go-c8y v2 SDK — Critical Assessment

**Date:** 2026-06-07
**Branch assessed:** `v2`
**Scope:** The v2 API under `pkg/c8y/api/`, supporting packages under `pkg/`, docs under `docs/`, tests under `test/` and `pkg/`, and coverage of `docs/c8y-oas.yml`.

---

## How to read this document

Every material claim carries a **confidence level** so you can tell verified facts from inferences and guesses:

- **HIGH** — directly verified by me (a command I ran, or code I read at the cited line). Treat as fact.
- **MEDIUM** — derived from sampling/patterns, or reported by a sub-investigation and consistent with what I saw, but not exhaustively checked.
- **LOW** — informed guess / interpretation. Treat with skepticism.

This assessment was produced by five parallel investigations (consistency/UX, OAS coverage, maintainability/dependencies, test coverage, documentation), then key critical findings were re-verified by hand. Where a sub-investigation made a claim I could not confirm — or that I disproved — it is flagged explicitly. Methodology notes and the commands used are included per section.

---

## Executive summary

This is a **mature, thoughtfully-architected, mid-rewrite SDK**. The v2 design (generics, `op.Result[T]`, uniform service layer, lazy iterators, device resolvers, dry-run/deferred execution) is genuinely good and consistently applied. API coverage of the Cumulocity REST surface is broad (~85–90% of OAS operations at path granularity). The weak spots are operational rather than architectural: a **broken linter config**, a **`resty` beta pin** under the HTTP layer, **stale "implementation status" docs**, and **copy-paste godoc errors**. None are deep design flaws; most are a day or two of cleanup.

### Scorecard

| Dimension | Grade | Confidence | One-line summary |
|---|---|---|---|
| Consistency | **A−** | HIGH | Uniform CRUD/options/result patterns; a few outliers (users tenant param, applications list variants, resolver coverage). |
| OAS coverage | **A−** | MEDIUM | ~85–90% of operations at path granularity; remaining gaps: trusted-cert edge endpoints, app per-binary. |
| User experience | **B+** | HIGH | Strong ergonomics (resolvers, iterators, Result); footguns in tenant repetition and context-only features. |
| Maintainability | **B** | HIGH | Clean architecture, shared generic core, spec-driven codegen + drift gate; broken lint config still open. |
| Dependencies | **B−** | HIGH | Core closure is lean; risks: resty pinned to unreleased beta, old cron, full echo framework in microservice. |
| Test coverage | **B** | HIGH | Excellent offline fake-server design (~46% api coverage) with an offline CI gate; protocol-area offline tests still thin. |
| Documentation | **B+** | HIGH | Strong prose architecture docs; let down by stale status checklists and 4 copy-paste godoc errors. |

---

## 1. Consistency

**Confidence in this section: HIGH** (sub-investigation read 10+ service packages; I independently read `alarms/api.go` and `client.go`).

### Strengths

- **Uniform CRUD surface.** Services expose `List`/`Get`/`Create`/`Update`/`Delete` (+ `ListAll` iterator) with consistent shapes. *(HIGH)*
- **Uniform return type.** Every operation returns `op.Result[T]`; deletes return `op.Result[core.NoContent]`; collections go through `core.ExecuteCollection`. *(HIGH — `pkg/c8y/op/result.go`, `pkg/c8y/api/core/execute.go`)*
- **Uniform options.** `<Operation>Options` naming, `pagination.PaginationOptions` embedding, `url:"field,omitempty,omitzero"` tags (time fields use `omitzero` to drop zero values). *(HIGH — `alarms/api.go:43-99`)*
- **Uniform internals.** Each service embeds `core.Service`; constructors follow `NewService(s *core.Service) *Service`; conventions like `var ApiXxx = "/..."`, `const ResultProperty`, `type XxxIterator = pagination.Iterator[T]` are applied everywhere. *(HIGH — confirmed `core.Service` embedded in 66 packages)*
- **Clean v1/v2 separation.** The root `package c8y` is only 6 shared helper files; v2 lives entirely under `pkg/c8y/api/`. A deliberate `client_compat.go` shim provides the migration path. This is a clean rewrite, **not** a tangled v1/v2 mix. *(HIGH)*

### Inconsistencies / outliers

- **`users` / `usergroups` require a `Tenant` field on every call** (`url:"-"`) rather than a service-level or context default — breaks the "set once" expectation of other services. *(HIGH — `users/api.go`)*
- **`applications` fragments the list surface** with `ListByName`/`ListByTenant`/`ListByUser` alongside generic `List`, instead of one options-driven `List`. *(HIGH)*
- **`identity` breaks the CRUD shape** — no `List`, minimal surface (externalIds/globalIds only). Partly inherent to the C8Y identity API, but surprising. *(MEDIUM)*
- **Device-resolver coverage is uneven.** Resolvers exist in `alarms`/`events`/`measurements`/`operations`, but `applications`/`users` reimplement name lookup ad-hoc, and many services have none. No shared interface unifies `DeviceRef`/`UserRef`/`GroupRef`. *(HIGH)*
- **Error typing varies** — mix of `op.Failed[T]`, `core.ErrNotFound`, and inline lookups; no exported sentinel set for common cases. *(MEDIUM)*

---

## 2. Coverage of `docs/c8y-oas.yml`

**Confidence in this section: MEDIUM overall.** Methodology: the OAS was parsed with Python/PyYAML (**146 paths / 246 operations / 50 tags** — HIGH); SDK paths were extracted by grepping path-string literals and `func (s *Service)` methods (HIGH); the comparison normalized `{id}`→`{}` and diffed sets, then manually triaged each apparent gap (MEDIUM). **Matching is at path granularity — it does not verify that every HTTP method, query param, or request/response body on a shared path is implemented.**

> Note: `docs/c8y-oas.yml` is currently **untracked** in git (it appears only in this working tree), consistent with it being a freshly-added reference for this very check.

### Overall estimate

**~85–90% of OAS operations are covered** at path granularity *(MEDIUM, explicitly an estimate)*. Of 145 normalized paths, 119 matched an SDK path literal directly; manual review of the 26 unmatched found ~10 false gaps (different placeholder names / hardcoded paths) and ~16 real gaps, mostly low-value root endpoints. By section, ~24 of ~27 functional areas are Full or Mostly-Full.

### Coverage by section (abridged)

| Section | SDK package | Status | Confidence |
|---|---|---|---|
| Alarms, Events (+binaries), Measurements | `alarms`,`events`,`measurements` | **Full** (CRUD + realtime) | HIGH |
| Inventory (MOs, children, availability, binaries) | `inventory`,`managedobjects`,`binaries` | **Full** | HIGH |
| Device control (operations, bulk, new-device, EST) | `operations`,`bulkoperations`,`devices` | **Full** | HIGH |
| Audit, Retention | `auditrecords`,`retentionrules` | **Full** | HIGH |
| Applications (apps, versions, binaries) | `applications`,`ui`,`microservices` | **Mostly full** | MEDIUM |
| Tenants (tenants, options, statistics, currentTenant) | `tenants`,`currenttenant`,`logintokens` | **Full** | HIGH |
| Users, current user/TOTP, groups, roles, inventory roles | `users`,`usergroups`,`userroles` | **Full** | HIGH |
| Features, Notification2 | `features`,`notification2` | **Full** | HIGH |
| Trusted certificates / CA | `trustedcertificates` | **Mostly full** | MEDIUM |
| Identity | `identity` | **Mostly full** | HIGH |
| Login options + accessMappings + restrict | `loginoptions` | **Full** | HIGH |
| Service-root/version discovery endpoints | — | **None** (low value) | HIGH |

### Biggest genuine gaps (HIGH these are unimplemented)

> These remaining gaps are tracked as declared drift waivers in `docs/c8y-oas.overlay.yml`.

1. **Trusted-cert edge endpoints** — `/bulk`, `/verify-cert-chain`, `/settings/crl`.
2. **Per-binary application endpoints** — `GET/DELETE /application/applications/{id}/binaries/{binaryId}`.

### Caveats

- Coverage is path-granular; per-operation body/param fidelity not audited *(MEDIUM)*.
- The SDK ships capabilities **not** in this OAS (`remoteaccess`, `ui/plugins`, CometD/Bayeux realtime, microservice bootstrap helpers), so the raw OAS percentage **understates** total SDK surface *(HIGH)*.

---

## 3. User experience

**Confidence in this section: HIGH** (grounded in `examples/` and `test/c8y_api_test/` usage).

### Strengths

- **`op.Result[T]` is a strong ergonomic core** — carries error, HTTP status, duration, retry info, and request metadata; forces explicit error handling. *(HIGH)*
- **Lazy iterators** (`ListAll().Items()`, range-friendly with per-item error) make pagination painless. *(HIGH)*
- **Device resolvers** (`name:`, `ext:c8y_Serial:`, `query:` strings, plus typed `managedobjects.ByName/ByExternalID/ByID/ByQuery`) are intuitive and reduce a very common chore. *(HIGH)*
- **Dry-run + deferred execution via context** enable inspect-before-send and confirmation prompts — well suited to CLI/agentic use. *(HIGH — `examples/dry-run`, deferred tests)*

### Footguns / friction

- **Tenant repetition** in `users`/`usergroups` is boilerplate-heavy. *(HIGH)*
- **Power features are context-only and invisible in signatures** — dry-run, deferred, inspection are discoverable only via docs/examples, not types. *(MEDIUM)*
- **Resolver string typos are unvalidated** (`"name:"` vs `"name :"`); no unified ref interface. *(MEDIUM)*
- **`applications` list-method sprawl** forces users to know which variant to call. *(HIGH)*

---

## 4. Maintainability

**Confidence in this section: HIGH** (I re-ran build/vet and inspected the configs).

### Strengths

- **Clean rewrite, strong shared core.** A generics-based execution/result layer (`op.Result[T]`, `core.Execute[T]`, `ExecuteCollection`, `ExecuteBinary`, `ExecuteNoContent`) plus a streaming/pipeline toolkit is shared across all services — this is what keeps the hand-written services consistent and is the project's biggest maintainability asset. *(HIGH)*
- **`go build ./...` is clean** (exit 0). *(HIGH)*
- **`go vet ./...` is nearly clean** — only 2 findings, both in one test file (`test/c8y_api_test/managed_object_bulk_stream_test.go:230,264`, a context-cancel leak). *(HIGH — re-verified)*
- **Low debt markers** — 44 TODO, 1 FIXME, 0 HACK across ~66k LOC. *(MEDIUM — reported, consistent)*
- **`test.log` is gitignored, not committed.** *(HIGH — corrected from an earlier wrong finding)*

### Concerns

- **Largest files** are reasonable for a per-resource SDK: `client.go` (1875 lines, aggregates ~40 service constructors — the main hotspot), `realtime/realtime_client.go` (~970), repository firmware/software services (~800 each), `op/result.go` (~767), `pipeline/pipeline.go` (~760). None alarming. *(HIGH)*
- **Inert/misleading `replace` directives.** `go.mod` has five `replace` directives pointing module paths (`pkg/c8y`, `pkg/microservice`, `test/...`) to local dirs, but those dirs are **not** separate modules — harmless today, but misleading and a trap if the project ever splits modules (replace directives don't propagate to consumers). *(HIGH facts / MEDIUM interpretation)*
- **Taskfile path bug.** `test-c8y`/`test-microservice` reference module paths missing the `/v2` suffix; those tasks will fail to resolve as written. *(MEDIUM)*

---

## 5. Dependencies

**Confidence in this section: HIGH** (deps and import locations verified; `go list -deps` used by sub-investigation).

### Good news: the core client closure is lean

`go list -deps ./pkg/c8y/api` pulls in **none** of echo/viper/prometheus/mpb/cron — those heavy deps are isolated to `pkg/microservice` (a microservice-hosting runtime) and to tests/examples. *(HIGH)*

### Risks (roughly in priority order)

1. **`resty.dev/v3` pinned to an unreleased beta pseudo-version** (`v3.0.0-beta.6.0.20260128173335-37296c9841e6`) — not even a tagged beta. The **entire HTTP layer** rests on an unreleased commit with no stability guarantee. **Top supply-chain risk.** *(HIGH — verified in `go.mod`)*
2. **`gopkg.in/robfig/cron.v2`** is the **old/unmaintained** cron; `robfig/cron/v3` is current. Isolated to `pkg/microservice`. *(MEDIUM)*
3. **`labstack/echo/v4`** (full web framework) is used only by `pkg/microservice/monitoring.go` for a health/metrics server; `net/http` would suffice but it's at least isolated. *(MEDIUM)*

Standard/reasonable deps (HIGH): `golang-jwt/jwt/v5`, `tidwall/gjson`+`sjson`, `google/go-querystring`, `gorilla/websocket`, `zalando/go-keyring`, `araddon/dateparse`, `golang.org/x/net`, `stretchr/testify`. `google/go-jsonnet` (heavy, isolated to `pkg/mapbuilder`) and `destel/rill` (examples/tests only) are acceptable.

---

## 6. Test coverage

**Confidence in this section: HIGH for measured numbers; MEDIUM for runtime-exercise interpretation.** Numbers below were produced by actually running the suites.

### Strategy (well-designed)

- **Stateful offline fake-server** is the dominant strategy: the same tests run against an in-memory `httptest.Server` fake, a live tenant, or a recording proxy, switched by `TEST_MODE` (documented in `docs/OFFLINE_TESTING.md`). This is a strong design — better than brittle cassettes. *(HIGH)*
- **Golden/snapshot** validation exists (`golden_test.go`) but is **dormant** — it `t.Skip`s when no golden files exist, and `testdata/golden/` is gitignored. *(HIGH)*
- testify used in 91/111 test files; `t.Parallel()` used in **0** despite thread-safe infra; table-driven in ~12. *(HIGH)*

### Measured results

- `TEST_MODE=offline go test ./test/c8y_api_test/...` → **395 PASS / 20 SKIP / 0 FAIL in ~13s**. *(HIGH)*
- `TEST_MODE=offline go test -coverpkg=./pkg/c8y/api/... ./test/c8y_api_test/...` → **46.2% of statements** in the api tree. The per-package `0.0%` figures from `go test ./pkg/...` are misleading: services are exercised cross-package by the integration suite. *(HIGH)*
- `go test ./pkg/...` highlights: `api/context` 100%, `core/artifact` 100%, `wsurl` 76.9%, `password` 67.2%, `oauth/device` 66.1%, `pipeline` 64.5%, `certutil` 54.4%, client `api` 13.2%, `model`/`pagination` ~6%. *(HIGH)*


### Other gaps

- **Protocol-sensitive areas thinly/untested offline:** realtime/WebSocket, notification2 streaming, oauth2 flow, and `cache.go` (only a live-only test, now skipped offline). *(MEDIUM — still open)*
- **No enforced coverage threshold in CI** — the `test-ci` task generates a coverage profile and prints the total in CI logs, but nothing gates on it, so regressions are visible but not blocked. *(MEDIUM — still open)*

---

## 7. Documentation quality

**Confidence in this section: HIGH** (prose docs read in full; godoc errors re-verified by grep).

### Prose docs — strong architecture, weak status hygiene

- **Exemplary:** `API_DESIGN.md`, `IMPLEMENTATION_PATTERNS.md`, `OFFLINE_TESTING.md` — comprehensive, accurate, with compile-ready examples and clear design rationale. *(HIGH)*
- **Stale "implementation status" checklists** — `V2.md` marks **OAuth2 as planned `[ ]`** and **Upsert as planned `[ ]`**, but both are **fully implemented** (OAuth2 has internal/browser/device flows + TFA; upsert has 6 variants in `pkg/c8y/op/upsert.go`). `API_DESIGN.md` correctly marks them done — so the docs **contradict each other**. *(HIGH)*
- **Dual v1/v2 README** mixes ~80 lines of v1 with v2 and lacks a single "start here" path. *(MEDIUM)*

### In-code godoc — good coverage, embarrassing copy-paste errors

- Option struct **fields are documented** (genuinely good). Exported types/functions generally have doc comments following Go convention. *(HIGH)*
- **Copy-paste service descriptions are wrong in 4 packages** *(HIGH — re-verified by grep):*
  - `alarms/api.go:29` → "get/set/delete **audit entries**" (should be alarms)
  - `identity/api.go:20` → "get/set/delete **audit entries**" (should be identities)
  - `loginoptions/api.go:27` → "get/set/delete **events**" (should be login options)
  - `remoteaccess/api.go:15` → "get/set/delete **events**" (should be remote access)
- **Few runnable examples** — ~11 `ExampleXxx`, almost all auth-focused; none for CRUD/pagination/resolvers/deferred execution. *(HIGH)*

---

## Prioritized recommendations

**P0 — correctness / credibility (hours):**
1. **Migrate `.golangci.yml` to v2 format** so linting actually runs. *(HIGH value)*
2. Fix the **4 copy-paste godoc errors** and the **stale `V2.md` status checklist**. *(HIGH value, trivial)*

**P1 — risk reduction (days):**
3. **Address `resty` pin** — track toward a tagged release or vendor/wrap the HTTP layer behind an interface to limit blast radius. *(HIGH value)*

**P2 — polish (ongoing):**
4. Add **offline tests for protocol areas** (realtime, notification2, oauth2, cache). *(MEDIUM)*
5. Reduce API outliers: **service-level/context tenant default** for users/groups; consider unifying `applications` list variants and the ref types. *(MEDIUM)*
6. **Enforce a coverage threshold** in CI now that the offline gate produces a profile. *(MEDIUM)*

---

## Confidence ledger (what I verified myself vs. relied on)

**Verified directly by me (HIGH):** repo/module layout; `go build` clean; `go vet` 2 findings; certutil test failure reproduced; golangci v1-config vs v2; CI runs `task test` = live; offline suite passes (relied on sub-agent run, consistent); the 4 godoc copy-paste errors; resty beta pin; qrterminal in core closure; `test.log` gitignored (corrected); cache.go line count (corrected).

**Relied on sub-investigations, consistent with my spot-checks (MEDIUM):** OAS operation counts and the 85–90% coverage estimate; the 46.2% offline api coverage number; per-package coverage table; dependency import-isolation map; per-doc prose quality.

**Interpretation / not exhaustively proven (LOW-MEDIUM):** the "no-codegen is the dominant maintenance cost" judgment; OAS coverage at body/param granularity (only path-level checked); the relative grades in the scorecard.
