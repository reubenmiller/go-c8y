## Why

`microservice.TenantContext` never checks incoming credentials. It splits the tenant out of the Basic auth username, or reads the `ten` claim of a bearer token without verifying its signature. With `UseServiceUser: true`, it then runs the request as that tenant's service user. Anyone who can reach the container port can act as the service user of any subscribed tenant by sending `Authorization: Basic base64("t123/x:y")`. The Java SDK validates every request with `GET /user/currentUser`. Validation on every request is too expensive for busy microservices, so it must be cached and must have an opt-out. Review: docs/V2_STABILIZATION_REVIEW.md §4.1 and §4.2.

## What Changes

- New `TenantContextOptions.Validation` (`ValidateDefault`, `ValidateNone`, `ValidateCached`), `ValidationTTL`, and a custom `Validator` hook.
- **BREAKING** (behavior) `ValidateDefault` means `ValidateCached` when `UseServiceUser` is true. Service-user mode now rejects unverified callers unless the user opts out with `ValidateNone`. User-scope mode stays unvalidated by default, because the platform already checks the forwarded credentials on the downstream call.
- Cached validation: one `GET /user/currentUser` per distinct credential per TTL, shared by concurrent requests, bounded in size, with short negative caching.
- The validated caller (`CallerInfo`: tenant, user name, effective roles) is stored in the request context. `TenantFromContext` returns the verified tenant when validation ran.
- New `RequireRoles(roles ...string)` middleware, which always validates the routes it wraps.

Default confirmed by the owner (2026-11-19).

Security fix. Can land independently of the other changes, and before v2.0.0.

## Capabilities

### New Capabilities
- `microservice-request-auth`: how a microservice authenticates incoming requests, binds tenant or service-user credentials to the request context, and checks roles.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/microservice/requestcontext.go` (middleware, options, context data), new `pkg/microservice/validation.go`.
- `examples/microservice/multi-tenant`, `docs/MICROSERVICE_TENANT_CONTEXT.md`.
- Microservices using `UseServiceUser: true` make one extra platform call per distinct caller per TTL, unless they opt out.
