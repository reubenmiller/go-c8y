## Context

- `authFromRequest` reads Basic auth (`{tenant}/{user}:{password}`), a bearer token (tenant from the unverified `ten` claim), or the `authorization` cookie (browser sessions, with an XSRF check for state-changing methods).
- User scope (default): the caller's credentials are forwarded downstream; a bad credential fails on the first platform call. Service-user scope (`UseServiceUser: true`): the caller's credentials are never sent to the platform, so nothing ever checks them.
- The Java SDK checks `GET /user/currentUser` with the forwarded `Authorization` header (or cookie plus `X-XSRF-TOKEN`) on each request.
- Concern from review feedback: a check per request doubles platform calls, and many distinct users multiply the load.

## Goals / Non-Goals

**Goals:**
- Close the service-user impersonation hole by default.
- Keep platform load to one check per distinct credential per TTL.
- Let users opt out, or plug in their own check.

**Non-Goals:**
- Local JWT signature verification (no documented key endpoint to rely on; a custom `Validator` can do it).
- Permission checks beyond roles (inventory roles, device permissions).

## Decisions

**Validation mode with a mode-dependent default.** `ValidateDefault` resolves to `ValidateCached` in service-user mode and `ValidateNone` in user scope. In user scope the downstream call already validates, so a separate check would only add load. Alternative: always validate by default (Java SDK behavior). Rejected because of the load concern; users who want it set `ValidateCached` explicitly.

**Cache keyed by credential hash.** Key: SHA-256 of the `Authorization` header value, or of the cookie value for cookie sessions. Raw credentials are never stored.

**TTL capped by token expiry.** Default TTL 5 minutes. For bearer tokens and cookies, the entry expires at `min(now + TTL, exp claim)`, so a cached entry never outlives the token. The `exp` claim is read without verifying the signature; it only shortens the TTL, so a forged claim cannot extend access.

**Concurrent first requests share one call** with `golang.org/x/sync/singleflight`. `golang.org/x/sync` (v0.23.0) is already in the module graph, so this adds no new module. A burst from one user makes one platform call.

**Bounded memory and negative caching.** LRU with a default of 10,000 entries. Failed validations are cached for 10 seconds so bad credentials cannot hammer the platform.

**`CallerInfo` in the context.** Validation stores tenant, user name, and effective roles. `TenantFromContext` prefers the verified tenant. `RequireRoles` reads roles from it, so role checks add no calls.

**`RequireRoles` forces validation.** It wraps a handler and validates even when the surrounding middleware uses `ValidateNone`, because roles cannot be known otherwise.

**Platform errors during validation.** A `401` or `403` from `/user/currentUser` rejects the caller with `401`. A `5xx` or timeout returns `503` and is not cached, so a platform outage does not lock users out for the negative-cache period.

## Risks / Trade-offs

- [Revoked credential stays valid until TTL expiry] → Default TTL is short (5 minutes) and configurable; document it.
- [Extra load for services with very many distinct callers] → `ValidateNone` opt-out, `Validator` hook for shared caches (for example Redis across replicas).
- [Behavior change for existing `UseServiceUser` users] → Call it out in the CHANGELOG with the opt-out.

## Default decision

**2026-11-19: keep the proposed default.** `ValidateDefault` is `ValidateCached` in service-user mode and `ValidateNone` in user scope. `ValidateNone` stays available as the opt-out in service-user mode.

## Open Questions

- None at this stage.
