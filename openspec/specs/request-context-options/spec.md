# request-context-options Specification

## Purpose
Defines where request-scoped options (auth, tenant, dry run, deferred execution, processing mode, extra headers and query parameters, redaction) are attached to a `context.Context`: package `api` is the only public home, and the context keys are internal.
## Requirements
### Requirement: Single public package for request options
All public functions that attach request-scoped options to a `context.Context` SHALL be exported from package `github.com/reubenmiller/go-c8y/v2/pkg/c8y/api`. No other public package SHALL export setters for the same options.

#### Scenario: Processing mode from api
- **WHEN** a caller uses `api.WithProcessingModeTransient(ctx)` and creates an event with that context
- **THEN** the request carries the header `X-Cumulocity-Processing-Mode: TRANSIENT`

#### Scenario: Extra headers from api
- **WHEN** a caller uses `api.WithExtraHeaders(ctx, map[string]string{"X-Test": "1"})`
- **THEN** the request carries `X-Test: 1`

#### Scenario: Dry-run handler from api
- **WHEN** a caller uses `api.WithDryRunHandler(api.WithDryRun(ctx, true), fn)` and sends a request
- **THEN** `fn` receives the request and nothing is sent to the server

### Requirement: No public package named context
The module SHALL NOT contain a public package named `context`.

#### Scenario: Package list
- **WHEN** `go list ./pkg/...` is run
- **THEN** no listed package path ends in `/context`

### Requirement: Context keys are internal
The context keys and their low-level helpers SHALL live in an internal package, so that only the `api` functions are part of the public API.

#### Scenario: External import is rejected
- **WHEN** code outside the module imports `github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers`
- **THEN** the Go toolchain rejects the import

### Requirement: Only typed context keys
The SDK SHALL read request options only from unexported, typed context keys. It SHALL NOT read untyped string keys such as `"tenant"`.

#### Scenario: String key is ignored
- **WHEN** a caller uses `context.WithValue(ctx, "tenant", "t999")` and lists users without setting a tenant
- **THEN** the request uses the client's tenant, not `t999`

#### Scenario: Supported override still works
- **WHEN** a caller uses `api.WithTenant(ctx, "t999")` and lists users without setting a tenant
- **THEN** the request path contains `t999`

### Requirement: Public getters for user middleware
Package `api` SHALL export read access for options that user code needs to inspect: `IsDryRun`, `IsDeferredExecution`, `IsMockResponses`, `ShouldRedactHeaders`, `TenantFromContext`, and `ProcessingMode`.

#### Scenario: Getter reflects setter
- **WHEN** a context is created with `api.WithTenant(ctx, "t123")`
- **THEN** `api.TenantFromContext(ctx)` returns `"t123", true`

