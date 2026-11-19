## 1. Check consumers

- [ ] 1.1 Search go-c8y-cli for imports of `api/contexthelpers` and `api/context`; list the functions it uses
- [ ] 1.2 Confirm every function from 1.1 will be available from `api` or is internal-only by design

## 2. Public surface in api

- [ ] 2.1 Create `pkg/c8y/api/context_options.go` and move the existing `api.With*` / `api.Is*` helpers into it
- [ ] 2.2 Add the missing setters: processing mode (5 functions), `WithExtraHeaders`, `WithExtraQueryParams`, `WithNoAccept`, `WithNoAuth`, `WithDryRunHandler`
- [ ] 2.3 Add the getters `TenantFromContext` and `ProcessingMode`
- [ ] 2.4 Move the package documentation from `api/context/doc.go` into the `api` godoc

## 3. Move and delete

- [ ] 3.1 Move `pkg/c8y/api/contexthelpers` to `pkg/c8y/internal/contexthelpers` and update all internal imports
- [ ] 3.2 Delete `pkg/c8y/api/context`, moving its tests to the `api` package
- [ ] 3.3 Remove the `ctx.Value("tenant")` string-key lookup from `MiddlewareRemoveEmptyTenantID`; check go-c8y-cli does not set it

## 4. Tests and docs

- [ ] 4.1 Offline tests for each scenario in `specs/request-context-options/spec.md`
- [ ] 4.2 Update `docs/V2.md`, `docs/DEFERRED_EXECUTION.md`, `docs/REQUEST_INSPECTION.md`, and examples
- [ ] 4.3 Add a migration note (old import → `api` function) to the CHANGELOG
