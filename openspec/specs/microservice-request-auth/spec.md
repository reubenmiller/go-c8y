# microservice-request-auth Specification

## Purpose
Defines how a microservice authenticates incoming requests (optional cached validation with the platform), binds tenant or service-user credentials to the request context, and checks roles.
## Requirements
### Requirement: Validation modes
`TenantContextOptions.Validation` SHALL support `ValidateDefault`, `ValidateNone`, and `ValidateCached`. `ValidateDefault` SHALL behave as `ValidateCached` when `UseServiceUser` is true, and as `ValidateNone` otherwise.

#### Scenario: Service-user mode rejects unverified callers by default
- **WHEN** a request with `Authorization: Basic base64("t123/x:wrong")` reaches a handler wrapped with `TenantContext(TenantContextOptions{UseServiceUser: true})` and `/user/currentUser` returns `401`
- **THEN** the middleware responds `401` and the handler is not called

#### Scenario: Opt-out
- **WHEN** the same request reaches a handler wrapped with `UseServiceUser: true` and `Validation: ValidateNone`
- **THEN** no `/user/currentUser` call is made and the handler runs with the tenant's service-user credentials

#### Scenario: User scope is not validated by default
- **WHEN** a request reaches a handler wrapped with `TenantContext()` using default options
- **THEN** no `/user/currentUser` call is made

### Requirement: Cached validation
With `ValidateCached`, the SDK SHALL call `GET /user/currentUser` with the caller's credentials at most once per distinct credential per TTL. Concurrent first requests with the same credential SHALL share one call. The cache SHALL be keyed by a SHA-256 hash of the credential and SHALL NOT store raw credentials.

#### Scenario: Repeated requests from one user
- **WHEN** one user sends 100 requests within the TTL
- **THEN** the fake server receives exactly one `/user/currentUser` request

#### Scenario: Concurrent burst
- **WHEN** one user sends 20 requests at the same time before any validation is cached
- **THEN** the fake server receives exactly one `/user/currentUser` request

#### Scenario: Distinct users
- **WHEN** 3 different users each send 10 requests within the TTL
- **THEN** the fake server receives exactly three `/user/currentUser` requests

### Requirement: Cache expiry
A cached validation SHALL expire at the earlier of `ValidationTTL` (default 5 minutes) and the bearer token's `exp` claim. Failed validations SHALL be cached for 10 seconds. Validation attempts that end in a `5xx` response or a timeout SHALL NOT be cached.

#### Scenario: Token expires before TTL
- **WHEN** a bearer token with `exp` 30 seconds away is validated and the same token is used 60 seconds later
- **THEN** the SDK validates it again

#### Scenario: Platform outage
- **WHEN** `/user/currentUser` returns `503`
- **THEN** the middleware responds `503`, and the next request with the same credential triggers a new validation

### Requirement: Verified caller in context
After successful validation, the request context SHALL carry `CallerInfo` (tenant, user name, effective roles), available through `microservice.CallerFromContext(ctx)`. `TenantFromContext` SHALL return the verified tenant from the platform response, not the tenant parsed from the header.

#### Scenario: Tenant comes from the platform
- **WHEN** a Basic auth header names tenant `t123` and `/user/currentUser` succeeds for a user of tenant `t123`
- **THEN** `TenantFromContext(r.Context())` returns `t123` and `CallerFromContext` returns the user's roles

### Requirement: Custom validator
`TenantContextOptions.Validator`, when set, SHALL replace the built-in `/user/currentUser` check while keeping the cache.

#### Scenario: Custom validator
- **WHEN** a `Validator` that accepts every request and returns tenant `t1` is configured with `ValidateCached`
- **THEN** no `/user/currentUser` call is made and `TenantFromContext` returns `t1`

### Requirement: Role checks
`RequireRoles(roles ...string)` SHALL return middleware that validates the caller (even when the surrounding `TenantContext` uses `ValidateNone`) and responds `403` unless the caller has all listed roles.

#### Scenario: Missing role
- **WHEN** a validated caller without `ROLE_INVENTORY_ADMIN` calls a route wrapped with `RequireRoles("ROLE_INVENTORY_ADMIN")`
- **THEN** the middleware responds `403` and the handler is not called

#### Scenario: Role present
- **WHEN** a validated caller with `ROLE_INVENTORY_ADMIN` calls the same route
- **THEN** the handler runs

