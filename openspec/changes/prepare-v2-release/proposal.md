## Why

Everything exported at v2.0.0 must be supported for the whole major version, and the first impression of v2 comes from the README. Today the README documents the v1 API (`github.com/reubenmiller/go-c8y/pkg/c8y`, `c8y.NewClientFromEnvironment(nil, false)`) and says there are no API stability guarantees. `docs/V2.md` and `docs/DEFERRED_EXECUTION.md` use import paths without `/v2`. Six v1 helper files in `pkg/c8y` and the `api/client_compat.go` shim are exported without a decision on whether they are supported v2 API. Nothing in CI stops a later v2.x release from breaking the API by accident. Review: docs/V2_STABILIZATION_REVIEW.md §5.4, §5.8, §5.9.

## What Changes

- Rewrite the README for v2: module path, quick start with `api.NewClient`, minimum Go version (1.27), stability promise, and links to the design docs.
- Fix import paths in all docs to include `/v2`.
- **BREAKING** Decide the status of `pkg/c8y/*.go` (cache, context, middleware, proxyreader, response, utils) and `api/client_compat.go` (`SendRequest`, `RequestResult`): document each as supported v2 API, or move it to `internal`.
- Document how `applications.ListByName`, `ListByTenant`, `ListByOwner`, and `ListByUser` differ from `List` with the matching filter, and that `Tenant` on tenant-scoped options defaults to the client tenant.
- Add an API compatibility check to CI (`gorelease` or `apidiff`) that compares each PR against the latest v2 tag.
- Release checklist: all must-land changes merged, spec vendored again, CHANGELOG grouped by area with migration tables, `v2.0.0` tag.

This is the last change before tagging.

## Capabilities

### New Capabilities
- `release-readiness`: what must be true of docs, exported packages, and CI before v2.0.0 is tagged, and how API compatibility is protected afterwards.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `README.md`, `CHANGELOG.md`, `docs/V2.md`, `docs/DEFERRED_EXECUTION.md`, `docs/API_DESIGN.md`.
- `pkg/c8y/*.go`, `pkg/c8y/api/client_compat.go`.
- `.github/workflows/v2.yml` (new compatibility job).
