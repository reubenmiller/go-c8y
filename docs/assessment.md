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

**Two corrections to sub-investigation claims, made during verification (both HIGH):**
1. `test.log` (564 KB) is **NOT committed** — it is gitignored via `*.log` (`git check-ignore test.log` → matched). An earlier finding claiming it was tracked is wrong.
2. `pkg/c8y/cache.go` is **365 lines**, not ~9500. The 9503 figure was a byte count misread as lines.

---

## Executive summary

This is a **mature, thoughtfully-architected, mid-rewrite SDK**. The v2 design (generics, `op.Result[T]`, uniform service layer, lazy iterators, device resolvers, dry-run/deferred execution) is genuinely good and consistently applied. API coverage of the Cumulocity REST surface is broad (~85–90% of OAS operations at path granularity). The weak spots are operational rather than architectural: **CI gates on a live tenant instead of the excellent offline suite**, a **broken linter config**, a couple of **dependency-supply-chain risks**, **stale "implementation status" docs**, and **copy-paste godoc errors**. None are deep design flaws; most are a day or two of cleanup.

### Scorecard

| Dimension | Grade | Confidence | One-line summary |
|---|---|---|---|
| Consistency | **A−** | HIGH | Uniform CRUD/options/result patterns; a few outliers (users tenant param, applications list variants, resolver coverage). |
| OAS coverage | **A−** | MEDIUM | ~85–90% of operations at path granularity; gaps in loginOptions sub-resources, identity search, alarm upsert. |
| User experience | **B+** | HIGH | Strong ergonomics (resolvers, iterators, Result); footguns in `any` bodies, tenant repetition, context-only features. |
| Maintainability | **B** | HIGH | Clean architecture & shared generic core; but 66 hand-written services (no codegen) + broken lint config. |
| Dependencies | **B−** | HIGH | Core closure is lean; risks: resty pinned to unreleased beta, abandoned pkcs7 domain, old cron, qrterminal leak. |
| Test coverage | **B−** → **B** | HIGH | Excellent offline fake-server design (~46% api coverage); ~~not run in CI; 1 real failing test; utils untested~~ — offline CI gate added, failing test fixed, util packages tested (2026-06-07). |
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
| Identity | `identity` | **Partial** | HIGH |
| Login options + accessMappings + restrict | `loginoptions` | **Partial** | HIGH |
| Service-root/version discovery endpoints | — | **None** (low value) | HIGH |

### Biggest genuine gaps (HIGH these are unimplemented)

1. **Login-options sub-resources** — `accessMappings`, `inventoryAccessMappings` (~8 ops), `PUT .../restrict`. The largest cohesive missing area.
2. **`POST /identity/search`** — bulk external-ID search.
3. **`POST /alarm/alarms/upsert`** — alarm upsert (events has upsert; alarms does not).
4. **Trusted-cert edge endpoints** — `/bulk`, `/verify-cert-chain`, `/settings/crl`.
5. **Per-binary application endpoints** — `GET/DELETE /application/applications/{id}/binaries/{binaryId}`.

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

- **`any` / `AdditionalProperties` bodies** in some Create paths lose type safety for custom fragments. *(HIGH — `alarms/api.go:211-215`)*
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

- **No code generation despite an OAS being present.** ~66 service packages are **hand-written** (no `go:generate`, no `Code generated` markers, `tools/` has only a `.zshrc`). This is the dominant long-term maintenance cost and a drift risk against the API. Consistency is held by convention alone. *(HIGH)*
- **Broken linter config.** `.golangci.yml` is **v1 format** (`run:`, `linters-settings:` at top level) but golangci-lint is now v2.x, which rejects v1 configs; CI uses `version: latest`. **Lint in CI is effectively broken or running defaults.** *(HIGH — verified config format + CI pin)*
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
2. **`go.mozilla.org/pkcs7`** is on an **abandoned vanity domain** (Mozilla wound these down). Used in `pkg/certutil`. Migrate to a maintained fork (e.g. `github.com/smallstep/pkcs7`). *(MEDIUM-HIGH)*
3. **`gopkg.in/robfig/cron.v2`** is the **old/unmaintained** cron; `robfig/cron/v3` is current. Isolated to `pkg/microservice`. *(MEDIUM)*
4. **`mdp/qrterminal/v3` leaks into the core client closure** — `devices/enrollment/api.go` imports it and is reachable from `devices/api.go`, so **every consumer of the client pulls a terminal-QR-rendering library**. Enrollment should return the URL/payload and let the CLI render the QR. *(MEDIUM-HIGH — verified import location)*
5. **`labstack/echo/v4`** (full web framework) is used only by `pkg/microservice/monitoring.go` for a health/metrics server; `net/http` would suffice but it's at least isolated. *(MEDIUM)*

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
- `go test ./pkg/...` highlights: `api/context` 100%, `core/artifact` 100%, `wsurl` 76.9%, `password` 67.2%, `oauth/device` 66.1%, `pipeline` 64.5%, `certutil` 54.4% **(FAILS)**, client `api` 13.2%, `model`/`pagination` ~6%. *(HIGH)*

### The two biggest problems

1. **CI gates on a live tenant, not the offline suite.** ~~`.github/workflows/main.yml` runs `task test`, which is `test-c8y` + `test-microservice` — both **live** (need `C8Y_HOST`/credentials). The fast, green, 46%-coverage **offline** suite (`task test-offline`) is **never invoked by CI**. Consequence: forked/external PRs without secrets get red builds, and there is **no offline gate protecting `pkg/`**.~~ *(HIGH — re-verified workflow + Taskfile)* — **✅ RESOLVED 2026-06-07:** a `test-offline` CI job now runs `task test-ci` (unit tests + offline integration suite with coverage) on every push/PR without needing a tenant; the live `test` job is guarded by an `if:` so it only runs when secrets are present and no longer red-builds forks.
2. **A genuinely failing test:** ~~`pkg/certutil` `TestParsePublicKeysPEM` and `TestPublicKeysFromFile` **FAIL deterministically offline** — `ParsePublicKeysPEM` can't derive an ECDSA public key from an ECDSA private-key PEM (RSA path works).~~ A real code-or-test defect, uncaught because CI doesn't run `./pkg/...`. *(HIGH — I reproduced it)* — **✅ RESOLVED 2026-06-07:** root cause was `parseECPrivateKey` only handling SEC1 keys while `MakeEllipticPrivateKeyPEM` emits PKCS#8; added a PKCS#8 fallback (symmetric with the RSA path). Both tests now pass.

### Other gaps

- ~~**Live tests panic rather than skip** without credentials (`test/microservice_test` panics; `test/c8y_test` fails) — poor contributor experience.~~ *(HIGH)* — **✅ RESOLVED 2026-06-07:** `BootstrapApplication` now `t.Skip`s on missing config instead of panicking; the `c8y_test` cache tests skip via a `skipWithoutCredentials` guard.
- ~~**Pure utility packages have no tests at all:** `pkg/mapbuilder`, `pkg/matcher`, `pkg/encoding`, `pkg/jsonUtilities` — low-hanging, high-value.~~ *(HIGH)* — **✅ RESOLVED 2026-06-07:** added table-driven testify suites for all four (coverage: `matcher` 89.7%, `encoding` 97.2%, `jsonUtilities` 93.9%, `mapbuilder` 93.4%).
- **Protocol-sensitive areas thinly/untested offline:** realtime/WebSocket, notification2 streaming, oauth2 flow, and `cache.go` (only a live-only test, now skipped offline). *(MEDIUM — still open)*
- ~~**Coverage is unmeasured in CI** — the real ~46% only surfaces via a manual `-coverpkg` run; regressions are invisible.~~ *(MEDIUM)* — **✅ PARTIALLY RESOLVED 2026-06-07:** the `test-ci` task now generates a coverage profile and prints the total in CI logs (no enforced threshold yet, so regressions are visible but not gated).

> **Remediation summary (2026-06-07):** P0 testing items from this section are fixed — certutil ECDSA bug, offline CI gate + coverage, graceful skips for live tests, and unit tests for the four bare util packages. The `Taskfile` `/v2` path bug (section 4) was also fixed as part of wiring up CI. Remaining open: offline tests for protocol-sensitive areas (realtime/notification2/oauth2) and an enforced coverage threshold.

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

> **Status update (2026-06-07):** items 1, 2, the Taskfile/live-test halves of 8, and the util-package half of 9 are **done** (see section 6 remediation). Remaining P0: items 3 and 4.

**P0 — correctness / credibility (hours):**
1. ~~Fix the **failing `certutil` ECDSA test/code** (`pkg/certutil`).~~ **✅ DONE** — PKCS#8 fallback in `parseECPrivateKey`. *(HIGH value)*
2. ~~**Add `task test-offline` (and ideally a coverage gate) to CI** so PRs are protected without a live tenant.~~ **✅ DONE** — `test-offline` CI job runs `task test-ci` (offline suite + `./pkg/...` + coverage). *(HIGH value)*
3. **Migrate `.golangci.yml` to v2 format** so linting actually runs. *(HIGH value — still open)*
4. Fix the **4 copy-paste godoc errors** and the **stale `V2.md` status checklist**. *(HIGH value, trivial — still open)*

**P1 — risk reduction (days):**
5. **Address `resty` pin** — track toward a tagged release or vendor/wrap the HTTP layer behind an interface to limit blast radius. *(HIGH value)*
6. **Move `qrterminal` out of the core closure** (enrollment returns payload; CLI renders). *(MEDIUM)*
7. Replace **`go.mozilla.org/pkcs7`** and **`robfig/cron.v2`** with maintained equivalents. *(MEDIUM)*
8. Fix the **`go vet` context leak** *(still open)*; ~~the **Taskfile `/v2` path bug**; make **live tests skip (not panic)** without credentials~~ **✅ DONE**. *(MEDIUM)*

**P2 — polish (ongoing):**
9. ~~Add **unit tests for pure util packages** (mapbuilder, matcher, encoding, jsonUtilities)~~ **✅ DONE** and protocol areas (realtime, notification2, oauth2, cache) *(still open)*. *(MEDIUM)*
10. Reduce API outliers: **service-level/context tenant default** for users/groups; consider unifying `applications` list variants and the ref types. *(MEDIUM)*
11. **Evaluate partial codegen from the OAS** for model structs / endpoint scaffolding to cut the 66-hand-written-service maintenance burden. *(LOW-MEDIUM — larger effort, big payoff)*
12. Close the named **OAS gaps** (loginOptions sub-resources, identity search, alarm upsert) if relevant to consumers. *(MEDIUM)*

---

## Confidence ledger (what I verified myself vs. relied on)

**Verified directly by me (HIGH):** repo/module layout; `go build` clean; `go vet` 2 findings; certutil test failure reproduced; golangci v1-config vs v2; CI runs `task test` = live; offline suite passes (relied on sub-agent run, consistent); the 4 godoc copy-paste errors; resty beta pin; qrterminal in core closure; `test.log` gitignored (corrected); cache.go line count (corrected).

**Relied on sub-investigations, consistent with my spot-checks (MEDIUM):** OAS operation counts and the 85–90% coverage estimate; the 46.2% offline api coverage number; per-package coverage table; dependency import-isolation map; per-doc prose quality.

**Interpretation / not exhaustively proven (LOW-MEDIUM):** the "no-codegen is the dominant maintenance cost" judgment; OAS coverage at body/param granularity (only path-level checked); the relative grades in the scorecard.
