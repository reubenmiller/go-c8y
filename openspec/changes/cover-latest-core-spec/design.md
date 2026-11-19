## Context

- `c8ygen lint` normalizes path templates and compares the set of OAS paths with path literals found in `pkg/c8y/api`. Waivers in the overlay (`drift.ignoreMissing`, `drift.ignoreExtra`) are path patterns.
- With the latest spec, lint reports 6 missing paths. Operation-level comparison finds 9 new operations; `POST /tenant/trusted-certificates/settings/crl` is already covered in practice by `certificaterevocationlist.AddFile` (`PUT`, which the spec still accepts for file upload).

## Goals / Non-Goals

**Goals:**
- Every operation in the latest core spec is either implemented or waived with a reason.
- The drift gate catches new methods on existing paths.

**Non-Goals:**
- Generating service methods from the spec (the generator stays at the option-struct and model layer, see docs/API_GEN.md).

## Decisions

**Method-aware lint by recording the method next to each path literal.** The linter currently finds path strings. To know the method, it reads the `SetMethod(resty.Method…)` call in the same builder function (the `…B` functions follow one pattern). Alternative: an explicit overlay table of implemented operations. Rejected: it would drift from the code. Fallback: where the method cannot be inferred, the linter reports the path as "method unknown" instead of failing.

**Method-level waivers.** Overlay waivers accept an optional method prefix, for example `POST /tenant/trusted-certificates/settings/crl # covered by PUT file upload`. Existing path-only waivers keep matching all methods.

**Restricted roles as a tenants sub-package.** `tenants/restrictedroles` follows the existing sub-package pattern (`tenantoptions`, `systemoptions`). The tenant defaults to the client tenant like other tenant-scoped services.

**Lint stays non-strict for one release after switching to method level**, so new method-level drift can be triaged, then goes back to `--strict`.

## Risks / Trade-offs

- [Method inference misses unusual builders] → Report "method unknown" and add an overlay hint for those few cases.
- [Spec changes again before release] → Vendor once more as a task in prepare-v2-release.

## Open Questions

- Is `/notification/realtime` reached by the realtime client through a path literal the linter cannot see (WebSocket URL building)? Confirm before deciding between a waiver and an implementation.
