# Proposal: Spec-Driven Code Generation for the v2 SDK

**Status:** Draft / RFC
**Date:** 2026-06-07
**Addresses:** `assessment.md` §4 ("No code generation despite an OAS being present") and rec. #10.

---

## 1. Problem

~62 service packages under `pkg/c8y/api/` are hand-written against `docs/c8y-oas.yml`
with **no generation step**. Consistency is held by convention alone, and there is no
mechanical guard against drift when the API changes. This is the dominant long-term
maintenance cost.

The naive fix — "run `oapi-codegen` over the OAS" — **is wrong for this SDK**, because
the SDK's public surface is *deliberately not* a 1:1 mirror of the API. Any proposal
must start from that fact.

---

## 2. The core tension: SDK surface ≠ API surface

The SDK intentionally diverges from the OAS to be more useful. A code generator that
overwrites these would destroy the SDK's value. The divergences fall into recurring
categories (all examples from `pkg/c8y/api/alarms/api.go`):

| # | Divergence | Example | In OAS? |
|---|---|---|---|
| D1 | **Resolver fields** — accept `name:`/`ext:`/`query:` strings, resolve to an ID before sending | `Source managedobjects.DeviceRef` | No — API takes a raw id |
| D2 | **Ergonomic input structs** — typed `CreateOptions` with `AdditionalProperties any` deep-merged into the body | `alarms.CreateOptions` | No — API takes a JSON body |
| D3 | **Synthesized operations** — endpoints the SDK adds on top of the raw API | `Upsert` + `markUpsertResult` Meta, `GetOrCreate*` | Partial / client-side |
| D4 | **Convenience list variants** | `applications.ListByName/ByTenant/ByUser` | One generic list op |
| D5 | **Cross-cutting wrappers** | `op.Result[T]`, `ListAll` iterators, dry-run/deferred, realtime `Subscribe*` | No |
| D6 | **Façade read models** — gjson accessors, not generated structs | `jsonmodels.Alarm.Severity()` | Schema exists, shape differs |
| D7 | **Param/field omissions & renames** | tenant handling, dropped root endpoints | API has more |

**Conclusion:** generation must target the *mechanical, 1:1-with-OAS* substrate and stop
at a stable seam. The ergonomic surface (D1–D5) stays hand-written, **composing** the
generated substrate rather than being replaced by it.

---

## 3. Design principles

1. **Generate the mechanical, hand-write the ergonomic.** The generator owns what is
   provably derivable from the spec; humans own judgment (resolvers, merges, naming).
2. **A hard, file-level seam.** Generated code lives in `zz_generated_*.go`
   (`// Code generated … DO NOT EDIT.`); hand-written code lives in `api.go` etc. They
   never share a file. Regeneration can never clobber human work.
3. **Composition, not inheritance of behavior.** Hand-written option structs *embed*
   generated ones; the hand-written `Service` *wraps* a generated raw layer.
4. **The spec is the source of truth — augmented by overlay annotations.** Divergence is
   declared *in the spec* via `x-` vendor extensions, not hidden in Go.
5. **Drift is caught in CI, not in review.** A check fails the build when the spec and the
   generated layer disagree.
6. **Incremental & reversible.** Adopt one resource at a time; a half-migrated repo
   always builds.

---

## 4. Proposed architecture: two layers

```
                OAS (docs/c8y-oas.yml)  +  overlay (x-c8y-sdk-* extensions)
                                 │
                          ┌──────┴───────┐  go generate ./...
                          │  generator   │
                          └──────┬───────┘
                                 ▼
  Layer 0 — GENERATED  (zz_generated_*.go, DO NOT EDIT)
    • path constants                     → zz_generated_paths.go
    • raw query/param option structs     → zz_generated_options.go   (url:"…,omitempty")
    • façade model accessors             → zz_generated_models.go     (jsondoc.Facade)
    • enums (string consts)              → zz_generated_enums.go
    • raw B-request builders + a rawService with thin 1:1 methods
                                 ▲ embedded / wrapped by
  Layer 1 — HAND-WRITTEN  (api.go — unchanged style)
    • Service{ rawService; DeviceResolver }   wraps Layer 0
    • ListOptions{ GenAlarmListParams; Source DeviceRef }   embeds Layer 0
    • resolver logic, CreateOptions+merge, Upsert, iterators, realtime, list variants
```

### The bridge contract (how D1–D6 are preserved)

- **D1/D2 (resolvers, ergonomic input):** the hand-written `ListOptions` **embeds** the
  generated param struct and adds resolver/ergonomic fields. The generated struct carries
  the boring `url:"…"` query params; the human adds `Source DeviceRef` and the resolve
  step. Nothing the human wrote is regenerated.

  ```go
  // zz_generated_options.go  (generated)
  type GenAlarmListParams struct {
      CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`
      Status      []model.AlarmStatus `url:"status,omitempty"`
      // …every query param from the OAS…
      pagination.PaginationOptions
  }

  // api.go  (hand-written)
  type ListOptions struct {
      GenAlarmListParams                       // ← generated query params
      Source managedobjects.DeviceRef `url:"source,omitempty"` // ← ergonomic override
  }
  ```

- **D3/D4/D5 (synthesized ops, variants, wrappers):** live only in `api.go`. The generator
  emits a raw `Create`/`List` returning `op.Result[T]`; `Upsert`, `GetOrCreate*`,
  `ListByName`, `ListAll`, `SubscribeStream` are hand-written and call the generated
  raw methods or B-builders.

- **D6 (façade models):** the generator emits `jsonmodels` accessors directly from schema
  `properties` (type → `.String()/.Int()/.Time()/.Bool()`), matching the existing
  `Facade` idiom. Hand-written constructors like `NewAlarmWithType` and any
  non-derivable accessor stay in a sibling non-generated file.

- **D7 (omit/rename):** controlled by overlay extensions (next section), never by editing
  generated output.

---

## 5. Governing divergence with overlay annotations

Divergence is declared **in the spec**, so the generator stays deterministic and the
intent is reviewable. The OAS **already carries codegen extensions**
(`x-codegen-resource-name` ×225, `x-codegen-ignore` ×11, `x-additionalPropertiesName`) —
likely feeding the sibling `go-c8y-cli` generator. We **reuse those where they exist** and
add a small, SDK-specific set:

| Extension | Scope | Effect |
|---|---|---|
| `x-codegen-resource-name` *(existing)* | operation/tag | Go package / type naming |
| `x-codegen-ignore` *(existing)* | operation | Skip entirely |
| `x-c8y-sdk-skip` | operation | Generate nothing (hand-written only) |
| `x-c8y-sdk-raw-only` | operation | Emit Layer-0 raw method + B-builder, **no** public method (human curates) |
| `x-c8y-sdk-resolver` | parameter/property | Emit field type as a resolver ref (e.g. `managedobjects.DeviceRef`) |
| `x-c8y-sdk-rename` | param/field | Override generated Go identifier |

To avoid bloating the upstream OAS, these can live in a **separate overlay file**
(`docs/c8y-oas.overlay.yml`, [OpenAPI Overlay 1.0](https://spec.openapis.org/overlay/v1.0.0.html))
merged at generation time — keeping the vendored spec pristine and the SDK's opinions in
one auditable place.

---

## 6. Generator implementation

**Recommendation: a small bespoke generator, not `oapi-codegen`/`openapi-generator`.**
Off-the-shelf generators emit concrete request/response structs and their own client
shape — fundamentally at odds with `op.Result[T]`, `jsondoc.Facade`, the `url:"…"` option
convention, and the `B`-suffix builder split. Bending their templates that far costs more
than owning a focused emitter.

- **Parse** with a mature library (`pb33f/libopenapi` or `getkin/kin-openapi`) so `$ref`
  (2,012 uses) and `allOf` (83 uses) resolution is not reinvented.
- **Emit** with `text/template`, one template per artifact (paths, options, models, enums,
  raw service). Run `goimports`/`gofmt` on output.
- **Live in** `tools/c8ygen/` (currently `tools/` holds only a `.zshrc`), wired via a
  repo-root `//go:generate go run ./tools/c8ygen` and a `task generate`.
- **Size estimate:** ~800–1,500 LOC. Templates mirror existing `api.go`/`jsonmodels`
  output, so generated code is byte-comparable to today's hand-written substrate.

### What is NOT generated (stays human)
Resolver resolution bodies, `*WithOptions`/merge logic, `Upsert` semantics, `GetOrCreate*`,
iterators, realtime, list-variant sugar, anything behind `x-c8y-sdk-skip`/`raw-only`.

---

## 7. Drift detection (ships first, cheap, high value)

Even before full generation, add a **read-only drift check** — this captures most of the
risk for a fraction of the effort:

`task lint-api` (run in CI) parses the OAS and the SDK and reports:
- OAS operations with **no** matching SDK path constant (missing coverage).
- SDK path constants with **no** matching OAS path (stale / typo, e.g. resolver typos).
- Per-operation **query params in OAS missing** from the corresponding option struct.
- Operations not covered and not explicitly waived via `x-c8y-sdk-skip` (so gaps are
  *declared*, not silent).

Output is a diff; exit non-zero on undeclared drift. This is the same parsing core the
generator uses, so it is a stepping-stone, not throwaway work.

---

## 8. Rollout plan

| Phase | Deliverable | Status |
|---|---|---|
| **0** | OAS parser + `task lint-api` drift report (non-gating) | ✅ **done** |
| **1** | Generator (`tools/c8ygen`); emit `zz_generated_paths.go` + `zz_generated_enums.go` into `pkg/c8y/api/spec` | ✅ **done** |
| **2** | Generate option struct + façade model for the **pilot** resource (`alarms`); refactor `api.go`/`jsonmodels` to compose them. Prove the seam. | ✅ **done** |
| **3** | Replace the in-code registry with a spec overlay (`docs/c8y-oas.overlay.yml`); roll the pattern across resources, resource-by-resource | 🔄 **in progress** — overlay + `extraFields` done; 7 resources migrated (`alarms`, `events`, `measurements`, `operations`, `tenants`, `auditrecords`, `binaries`); remainder is divergent or low-value (see below) |
| **4** | Make `task lint-api` **gating** (`--strict`) in CI; add `CONTRIBUTING` note: edit the spec/overlay, not `zz_generated_*.go` | ✅ **done** |

Each phase leaves the repo building and tested (the offline suite in
`OFFLINE_TESTING.md` is the regression net). Stop after any phase and still net positive.

### What Phases 0–1 delivered

- **`tools/c8ygen`** — a dependency-light generator (stdlib + `gopkg.in/yaml.v3`, already
  an SDK dependency; no heavy OpenAPI toolkit). Subcommands: `generate`, `lint`, `fetch`.
- **Spec on demand** — reads the vendored `docs/c8y-oas.yml` by default, or downloads the
  latest from `https://cumulocity.com/api/core/dist/c8y-oas.yml` with `--fetch`
  (`task fetch-spec` to vendor it; `generate --fetch` to build straight from upstream).
- **Generated `pkg/c8y/api/spec`** — 146 path constants + 52 enum groups, deterministic,
  `gofmt`-clean, marked `DO NOT EDIT`. Wired to `go generate` and `task generate`.
- **Drift check** (`task lint-api`) — already earning its keep: it reproduced the
  assessment's named OAS gaps (loginOptions accessMappings, `identity/search`, trusted-cert
  `bulk`/`verify-cert-chain`) **and** surfaced a dead, typo'd path constant in `binaries`
  (`/inventory/managedObject/{id}`, singular). CI runs it non-gating and verifies the
  generated substrate is committed and current.

> **Note on counts:** the generator extracts **146** paths from the spec (vs the
> assessment's manually-counted 145/146); both round-trip to the same coverage picture.

### What Phase 2 delivered (pilot: `alarms`)

- **Generated option struct** — `c8ygen resources` emits `pkg/c8y/api/alarms/zz_generated_options.go`
  containing the full `ListOptions` (all GET `/alarm/alarms` query params, typed enum
  slices, the `Source` resolver field, embedded pagination). The hand-written `api.go`
  no longer declares the struct; its `List` method keeps the resolver *behavior*.
- **Generated façade model** — `pkg/c8y/jsonmodels/zz_generated_alarm.go` holds the
  derivable scalar accessors (`ID`, `Severity`, `Time`, …). The hand-written `alarm.go`
  keeps only the non-derivable `SourceID()` (nested object) and the constructors.
- **Behaviour-preserving** — the full offline suite (incl. the alarms-heavy integration
  tests) passes unchanged. The generated output is byte-identical to the previous
  hand-written surface.
- **Driver** — an in-code `resources.go` registry declares the seam per resource
  (option type, field-type/doc overrides, embeds). This is the **precursor to the
  `x-c8y-sdk-*` overlay**; Phase 3 reads the same intent from the spec instead.

#### Design finding: generate the whole struct, don't embed a params sub-struct

`API_GEN.md` originally sketched the hand-written `ListOptions` *embedding* a generated
`GenListParams`. Implementing it surfaced a hard Go constraint: **promoted fields cannot
be set in a composite literal** — `alarms.ListOptions{Severity: …}` stops compiling once
`Severity` lives in an embedded struct, a breaking change for every caller. So the seam
for option structs is instead:

- **Layer 0 owns the whole struct's *shape*** (generated, including the `Source` field
  typed via override and the embedded `pagination.PaginationOptions`).
- **Layer 1 owns *behavior*** (the resolver step in `List`, `Upsert`, iterators, …).

Type/doc **overrides** in the registry (later: overlay) express the deliberate
divergences — `source → managedobjects.DeviceRef`, `status → []model.AlarmStatus` — so
the generated struct is API-compatible with the hand-written one it replaced. Façade
**models** compose the opposite way (generated methods + hand-written methods on the same
type), which has no literal-initialization constraint.

### What Phase 3 delivered (so far)

- **The overlay file is real** — the in-code registry is gone; `tools/c8ygen` now reads
  `docs/c8y-oas.overlay.yml`. The overlay declares, per resource, the option structs
  (path/method, field type & doc overrides, embeds, imports) and façade models (schema,
  skipped props). It is **kept separate from `c8y-oas.yml` so it survives `task fetch-spec`**.
  Implemented as a pragmatic operation-keyed overlay, not full OpenAPI Overlay 1.0 JSONPath
  (a possible future step); the structure mirrors the generator's model for clarity.
- **A second resource proves it generalizes** — `events` migrated to generated
  `ListOptions` + façade model by adding ~30 overlay lines and slimming its hand-written
  files to the resolver field / `SourceID()` / constructors. Byte-identical to the prior
  surface; the events integration tests pass unchanged.
- **Adding a resource is now a docs change**, not a Go change: append an entry to the
  overlay, run `task generate`, delete the superseded hand-written struct/accessors.

**Remaining rollout is intentionally incremental** — each resource needs its hand-written
option struct and model slimmed and verified behaviour-identical, so it should land
resource-by-resource (ideally maintainer-reviewed) rather than in one sweep.

#### Migration triage

The migration invariant is **behaviour-identical**: the generated `ListOptions` must have
exactly the same field set as the hand-written one (the generator emits only OAS params,
so a hand-written field absent from the OAS would be silently dropped — a breaking change).
Triaging the remaining resources against that invariant:

- **Migrated (7):** `alarms`, `events`, `measurements`, `operations` (resolver/enum-heavy —
  the highest-value cases), `tenants` (plain, options-only), and `auditrecords` + `binaries`
  (resolved below).
- **Resolved non-OAS-field cases:** `auditrecords.Revert` is a real server parameter the
  vendored OAS omits — kept via the overlay's **`extraFields:`** directive. `binaries.Text`
  was a copy-paste from the inventory options (unsupported by the endpoint) and was dropped.
- **Skipped — deliberately curated / would expand the public API:** `inventory/managedobjects`
  (6 curated fields + embedded `GetOptions` over ~17 OAS params), `applications`
  (`ListByName/ByTenant/ByUser` variants), `users` (per-call tenant). These intentionally
  expose a different surface than the raw OAS.
- **Skipped — pagination-only:** `bulkoperations`, `retentionrules`, `userroles`,
  `*options`, `*/versions`, … — nothing to generate beyond the pagination embed.
- **Not yet evaluated:** `repository/*` (firmware/software), `microservices`, `notification2`,
  `loginoptions`, `trustedcertificates`, `usergroups`, statistics sub-resources.

Everything remaining is a value/scope judgement rather than a generator limitation: the
`extraFields:` directive now lets a generated struct carry params the spec omits, so
"field not in the OAS" is no longer a blocker.

### What Phase 4 delivered

- **The drift check is a CI gate.** `task lint-api -- --strict` (the `api-drift` job) fails
  on any OAS↔SDK drift **not** declared in the overlay, so new endpoints, typos, and
  accidental removals are caught on every PR.
- **Declared waivers.** The overlay's `drift:` section enumerates the known-acceptable
  drift — service-root/discovery endpoints, non-OAS features (`/meta/*` realtime,
  `/service/remoteaccess/*`), and the known coverage gaps from the assessment (kept under
  a `TODO` comment so they stay visible). 41 items are waived today; `lint` prints the
  count and lists only undeclared drift. Patterns match normalized paths with a `*` prefix
  wildcard.
- **One real wart removed.** The gate forced resolution of the dead, typo'd
  `binaries.ApiManagedObject` (`/inventory/managedObject/{id}`, singular) the drift check
  first surfaced — deleted rather than waived.
- **[CONTRIBUTING.md](../CONTRIBUTING.md)** documents the contract: edit the spec/overlay
  and the hand-written layer, never `zz_generated_*.go`; run `task generate`; how to add a
  resource and record a drift decision.

---

## 9. Risks & trade-offs

- **Generator becomes its own maintenance burden.** Mitigated by keeping it small and
  scoped to the mechanical substrate; the hard/changing parts stay hand-written where
  flexibility already lives.
- **OAS inaccuracies propagate.** Mitigated because generation targets only path/param/enum/
  schema facts (high-fidelity in the spec) and the drift check surfaces mismatches; bodies
  and ergonomics — where the OAS is weakest — remain human.
- **Two-layer indirection.** A reader of `ListOptions` now follows an embed into generated
  code. Mitigated by the strict file-naming convention and a short `CONTRIBUTING` section.
- **Overlay coupling to upstream OAS versions.** Mitigated by keeping overlay keyed on
  operationId/path and failing loudly when a referenced operation disappears.

---

## 10. Recommendation

Adopt the **two-layer, spec-driven hybrid**: a small bespoke generator owns the
mechanical Layer 0 (paths, options, façade models, enums, raw builders); the existing
hand-written Layer 1 keeps every ergonomic divergence (D1–D5) by *composing* Layer 0;
divergence is declared via overlay `x-c8y-sdk-*` extensions; and a CI drift check guards
the seam.

**Start with Phases 0–1** (drift check + path/enum generation). They are low-risk, deliver
immediate anti-drift value, reuse the same parsing core, and require no change to the
curated public API — validating the approach before committing to full model/option
generation.
