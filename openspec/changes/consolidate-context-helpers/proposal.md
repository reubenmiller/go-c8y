## Why

Request-scoped options are exposed from three public packages: `pkg/c8y/api` (7 setters, 4 getters), `pkg/c8y/api/contexthelpers` (the full set of ~30 functions), and `pkg/c8y/api/context` (processing mode only, forwarding to `contexthelpers`). The package named `context` shadows the standard library, so every caller needs an import alias. Users cannot tell which package is the supported one, and all three would be frozen at v2.0.0. Review: docs/V2_STABILIZATION_REVIEW.md §5.2.

## What Changes

- Package `api` becomes the single public place for request-scoped options. It gains the setters it does not have yet: processing mode, extra headers, extra query parameters, no-accept, no-auth, dry-run handler.
- **BREAKING** Delete `pkg/c8y/api/context`. Nothing in the repo imports it.
- **BREAKING** Move `pkg/c8y/api/contexthelpers` to `pkg/c8y/internal/contexthelpers`. Getters that users need (for example `IsDryRun`) stay available from `api`.
- **BREAKING** Stop reading the untyped string context key `ctx.Value("tenant")` in `MiddlewareRemoveEmptyTenantID`. Any package can set or collide with a string key; `api.WithTenant` is the supported way to override the tenant.
- Update docs and examples that use `c8yctx` or `contexthelpers`.

Must land before v2.0.0.

## Capabilities

### New Capabilities
- `request-context-options`: the public functions that attach request-scoped behavior (auth, tenant, dry run, deferred execution, processing mode, headers, redaction) to a `context.Context`, and where they live.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/c8y/api/context` (deleted), `pkg/c8y/api/contexthelpers` (moved to internal), `pkg/c8y/api` (new re-exports).
- Internal imports in `pkg/c8y/op`, `pkg/c8y/api/core`, and service packages change path.
- `docs/V2.md`, `docs/DEFERRED_EXECUTION.md`, `docs/REQUEST_INSPECTION.md`, the `context/doc.go` content (moved into `api` godoc), examples.
