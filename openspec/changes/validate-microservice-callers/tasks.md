## 1. Validation core

- [ ] 1.1 Add `CredentialValidation` and its constants, `ValidationTTL`, and `Validator` to `TenantContextOptions`
- [ ] 1.2 Add `CallerInfo`, `CallerFromContext`, and make `TenantFromContext` prefer the verified tenant
- [ ] 1.3 Implement the built-in validator: `GET /user/currentUser` with the forwarded `Authorization` header, or cookie plus `X-XSRF-TOKEN`
- [ ] 1.4 Implement the cache: SHA-256 keys, TTL capped by `exp`, LRU limit, 10-second negative cache, no caching of `5xx`/timeouts
- [ ] 1.5 Share concurrent first validations with `golang.org/x/sync/singleflight`

## 2. Middleware

- [ ] 2.1 Apply the mode rules in `TenantContext` (`ValidateDefault` depends on `UseServiceUser`)
- [ ] 2.2 Map validation results to responses: `401` for rejected credentials, `503` for platform errors, through `OnError`
- [ ] 2.3 Add `RequireRoles` (forces validation, `403` on missing roles)

## 3. Tests

- [ ] 3.1 Offline tests for every scenario in `specs/microservice-request-auth/spec.md`, counting `/user/currentUser` calls on the fake server
- [ ] 3.2 Race test for the concurrent-burst scenario (`go test -race`)

## 4. Documentation

- [ ] 4.1 Update `docs/MICROSERVICE_TENANT_CONTEXT.md`: the security issue, modes, costs, and the opt-out
- [ ] 4.2 Update `examples/microservice/multi-tenant` to show `RequireRoles`
- [ ] 4.3 CHANGELOG entry flagging the behavior change for `UseServiceUser: true`
