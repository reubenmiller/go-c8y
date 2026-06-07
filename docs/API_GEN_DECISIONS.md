# API Codegen — Open Decisions

**Status:** awaiting decisions · **Branch:** `feat/api-codegen` · **Date:** 2026-06-07

The spec-driven generator ([API_GEN.md](API_GEN.md)) is built and 5 resources are
migrated. Further progress needs **product decisions**, not more mechanical work. This doc
lists each open decision with options and a recommendation. Tick a box (or note "Other")
and I'll implement.

---

## Decision summary

| # | Decision | Recommendation | Status |
|---|---|---|---|
| 1 | `auditrecords.Revert` — not in OAS | Verify against the live API, then drop or add to overlay | ✅ supported by server → added via `extraFields`; `auditrecords` migrated |
| 2 | `binaries.Text` — not in OAS | Drop (likely vestigial copy-paste) | ✅ dropped; `binaries` migrated |
| 3 | Add an overlay "extra hand-written fields" directive? | Only if #1/#2 say "keep" | ✅ `extraFields` directive built (needed for #1) |
| 4 | Curated resources (`managedobjects`, `applications`, `users`) | Keep curated — do **not** auto-expand | ✅ kept curated (documented in API_GEN.md triage) |
| 5 | Continue migrating the unevaluated batch (`repository/*`, `microservices`, …) | Yes, triage + migrate the clean ones | ✅ triaged; `notification2` + `loginoptions` migrated; rest are wrappers/path-param (kept) |
| 6 | Implement the waived **coverage gaps** (loginOptions sub-resources, `identity/search`, …) | Separate effort, prioritise `identity/search` + loginOptions | 🔶 `identity/search` **done**; loginOptions sub-resources **pending confirmation** (see note) |
| 7 | Push branch / open PR now, or after more migration? | Open PR now (infra is complete & self-contained) | ⏸ deferred — bundling into one PR per user |

> **#6 status.** `POST /identity/search` is implemented (`Identity.Search`). The
> **loginOptions sub-resources** (`accessMappings`, `inventoryAccessMappings`, `restrict`)
> are scoped but **not yet implemented**: 11 write-heavy operations (POST/PUT/DELETE) over
> **auth-configuration access mappings** — which govern who is granted which roles/apps on
> SSO login — plus 5 schemas, 2 new sub-packages and client wiring. Because it **mutates
> security-sensitive config and cannot be validated by the offline test harness**, it
> warrants explicit confirmation and ideally live-tenant validation before shipping. The
> remaining gaps (trusted-cert `bulk`/`verify-cert-chain`, app `binaries/{binaryId}`,
> `features/{featureKey}`, …) stay waived/deferred per the recommendation.

> **Resolved 2026-06-07 — #1, #2, #3.** `revert` is a real server parameter the vendored
> OAS omits, so it is declared via the new overlay `extraFields:` directive and
> `auditrecords` is migrated. `binaries.Text` was a copy-paste from the inventory options
> (unsupported by `GET /inventory/binaries`) and was dropped; `binaries` is migrated.
> **7 resources are now generated.** Decisions #4–#7 remain open.

Effort: S = <1h · M = a few hours · L = multi-day.

---

## 1. `auditrecords.Revert` — field not in the OAS

**Context.** `auditrecords.ListOptions` has a `Revert bool` field tagged `url:"revert"`,
with a hand-written comment: `// TODO: Check if this is supported or not`. The OAS GET
`/audit/auditRecords` has **no** `revert` parameter. The generator emits only OAS params,
so migrating auditrecords would silently **remove** `Revert` — a breaking change for any
caller setting it.

**Options.**
- [ ] **(A, recommended)** Verify against a live tenant whether `revert` actually works on
  `/audit/auditRecords`. If **no** → delete `Revert` (it's dead) and migrate the resource.
  If **yes** → it's an OAS gap; add the param to the overlay (see #3) and migrate.
- [ ] **(B)** Leave `auditrecords` hand-written indefinitely.
- [ ] **(C)** Drop `Revert` now without verifying (treat the TODO as confirmation it's bogus).

**Why A:** the TODO shows the original author was unsure; a 1-minute live check settles it
and either removes dead surface or documents a real gap.

---

## 2. `binaries.Text` — field not in the OAS

**Context.** `binaries.ListOptions` has a `Text string` field (`url:"text"`) whose comment
("Search for managed objects where a property value is equal…") is copied from the
inventory/managedObjects options. The OAS GET `/inventory/binaries` lists
`childAdditionId, childAssetId, childDeviceId, ids, owner, type` — **no** `text` param.

**Options.**
- [ ] **(A, recommended)** Drop `Text` (almost certainly a copy-paste that the binaries
  endpoint ignores) and migrate `binaries`.
- [ ] **(B)** Keep `Text` via the #3 "extra fields" directive (only if a live check shows
  the binaries endpoint honours `text`).

**Why A:** the comment is verbatim from a different resource and the endpoint doesn't
document the param; low risk it's actually used.

---

## 3. Add an "extra hand-written fields" overlay directive?

**Context.** If #1 or #2 resolve to "keep the field", the generator needs a way to add
fields that are **not** in the OAS to a generated option struct. Today it only emits OAS
params (plus type/doc overrides on those).

**Options.**
- [ ] **(A, recommended — conditional)** Build it *only if* #1/#2 need it. A small overlay
  block like:
  ```yaml
  extraFields:
    - name: Revert
      type: bool
      tag: "revert,omitempty"
      doc: "..."
  ```
  emitted after the OAS-derived fields.
- [ ] **(B)** Don't build it; resources needing extra fields stay hand-written.

**Trade-off:** the directive reintroduces a hand-maintained surface in the overlay, but
keeps the struct generated/consistent. Worth it only if ≥2 resources actually need it.

---

## 4. Curated resources — keep curated, or auto-expand to full OAS?

**Context.** `inventory/managedobjects` (6 curated fields + embedded `GetOptions`, vs ~17
OAS query params), `applications` (`ListByName`/`ByTenant`/`ByUser` variants), and `users`
(per-call `Tenant`) deliberately expose a **different, smaller/ergonomic** surface than the
raw OAS. Generating them as-is would either drop fields or balloon the public API.

**Options.**
- [ ] **(A, recommended)** Keep them hand-written/curated. This is the intended
  "API ≠ SDK" divergence; codegen should not flatten it.
- [ ] **(B)** Expand `managedobjects.ListOptions` to all OAS params (adds
  `childAssetId`, `onlyRoots`, `owner`, `withGroups`, `withLatestValues`, … — a public API
  addition, not a break, but changes the curated surface).
- [ ] **(C)** Migrate only the *models* for these (façade accessors), leaving the curated
  option structs hand-written. Lower-risk partial win.

**Why A:** the curation is a feature (see assessment §3 ergonomics). If you want broader
inventory query coverage, that's a separate API-design task, not codegen rollout.

---

## 5. Continue migrating the unevaluated batch?

**Context.** Not yet triaged: `repository/*` (firmware/software items/versions/patches),
`microservices`, `notification2`, `loginoptions`, `trustedcertificates`, `usergroups`,
and the statistics sub-resources. Each needs the same parity check (field set ==
OAS params) before migrating.

**Options.**
- [ ] **(A, recommended)** I triage all of them and migrate the ones with clean parity
  (same approach as alarms/events/…), skipping divergent ones with a documented reason.
- [ ] **(B)** Stop at the current 5 — the pattern is proven; treat the rest as ongoing
  maintenance done opportunistically.

**Why A:** likely a few more clean wins (repository/statistics resources tend to have
straightforward query params), and it shrinks the hand-written surface further.

---

## 6. Implement the waived coverage gaps?

**Context.** The drift gate waives known-missing endpoints in
`docs/c8y-oas.overlay.yml` under a `TODO` comment (from assessment §2):

- `POST /identity/search` (bulk external-ID search)
- loginOptions sub-resources: `accessMappings`, `inventoryAccessMappings`, `restrict` (~8 ops)
- trusted-cert `bulk`, `verify-cert-chain`
- `/application/applications/{id}/binaries/{binaryId}`, `applicationsByOwner`
- `/features/{featureKey}`, `/notification/realtime`, inventory `childDevices/{childId}`

These are **net-new SDK functionality** (hand-written services, not codegen).

**Options.**
- [ ] **(A, recommended)** Prioritise by consumer need — `identity/search` and the
  loginOptions sub-resources are the most-cited gaps; do those first, defer the rest.
- [ ] **(B)** Implement all of them.
- [ ] **(C)** Leave as documented gaps (they stay waived and visible).

**Note:** this is independent of the codegen work and can happen anytime.

---

## 7. Push the branch / open a PR now?

**Context.** The generator + drift gate (Phases 0–4) and 5 migrated resources are complete,
self-contained, and green. The branch `feat/api-codegen` is unpushed with 8 commits.

**Options.**
- [ ] **(A, recommended)** Open a PR now — the infrastructure is done and reviewable;
  further resource migration can be follow-up PRs.
- [ ] **(B)** Keep going on this branch (decisions #1–#6) and open one larger PR later.

**Why A:** the codegen infra is a coherent, complete unit; a focused PR is easier to review
than one that also churns many resources.

---

### How to respond

Reply with the picks, e.g. **"1A, 2A, 4A, 5A, 7A; defer 6"** — I'll implement in that order.
