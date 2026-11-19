# go-c8y v2: stabilization review

**Date:** 2026-11-19
**Branch reviewed:** `origin/v2` at `37d0690`
**Spec compared:** latest `https://cumulocity.com/api/core/dist/c8y-oas.yml` (downloaded 2026-11-19) against the vendored `docs/c8y-oas.yml` (vendored 2026-06-07)

This review covers four things: API coverage against the latest core spec, the four areas you named, and design changes worth making before the public API is frozen. Each item is tagged **Must** (do before v2.0.0, because it changes public API or is a security issue), **Should** (additive, can land in a minor release), or **Could**.

---

## 1. Coverage against the latest core OpenAPI spec

`go build ./...` and `go vet ./pkg/...` pass. With the vendored spec, `task lint-api` reports no undeclared drift (49 waived). With the latest spec it reports 6 missing paths. The linter compares paths only, so I also compared operations (method + path).

The latest spec adds **9 operations** (255 in total, up from 246):

| Operation | Method / path | SDK status |
|---|---|---|
| `postCertificateAccessToken` | `POST /tenant/oauth/certificate` | **Missing.** See §3. |
| `putCurrentApplicationResourceAuthType` | `PUT /application/currentApplication/authType` | **Missing.** Service user opts itself in to certificate auth. |
| `putApplicationResourceAuthType` | `PUT /application/applications/{id}/authType` | **Missing.** Admin resets a service user to `NONE`. |
| `getTenantApplicationRestrictedRolesResource` | `GET /tenant/tenants/{tenantId}/applications/restricted-roles` | **Missing** |
| `putTenantApplicationRestrictedRolesResource` | `PUT …/restricted-roles` | **Missing** |
| `postTenantApplicationRestrictedRoleResource` | `POST …/restricted-roles/{roleId}` | **Missing** |
| `deleteTenantApplicationRestrictedRoleResource` | `DELETE …/restricted-roles/{roleId}` | **Missing** |
| `postSmartRestTemplateCollectionResource` | `POST /inventory/templates/globalId/{globalId}/externalId` | **Missing** |
| `postCRLSettings` | `POST /tenant/trusted-certificates/settings/crl` | Covered in practice: `certificaterevocationlist.AddFile` uploads the file with `PUT`, which the spec still accepts. |

Schema changes in operations that already exist:

- `currentUser` gained `requiredAuthType` (`NONE` | `CERTIFICATES`). See §3.
- The application representation gained `applicationAuthType`.
- Bulk managed-object create and update now document `bulkRequestError` (`bulkRequest.failedIndex`, `bulkRequest.fragmentType`) and the `c8y_BatchId_*` recovery procedure. See §2.

Known gaps already waived in the overlay that are still open: `/notification/realtime`, `/tenant/tenants/{tenantId}/trusted-certificates/bulk`, `/tenant/trusted-certificates/verify-cert-chain`. The users `withCertificates` / `provisionedCertificates` support added in `1656d5d` is not in the public spec, so it relies on undocumented behavior.

**Actions**
- **Should:** vendor the new spec (`task fetch-spec`), implement the missing operations above, and waive or implement the rest so `lint-api --strict` stays green.
- **Could:** make `c8ygen lint` compare **method + path**. As it stands, a new verb on an existing path (for example `PUT /inventory/managedObjects`) is invisible to the drift gate.

---

## 2. Bulk managed-object create and update

**What exists:** `ManagedObjects.CreateBulk`, `UpdateBulk`, `CreateBulkStream`, and `UpdateBulkStream` (`pkg/c8y/api/inventory/managedobjects/service.go`). They use the right media type, and the stream variants batch into groups of 50 (`DefaultBulkBatchSize`). `model.ManagedObject` already has `ExternalIDs []Identity \`json:"externalIds"\``, and the bodies are `any`, so `externalIds` can be passed on create today.

**Gaps**

1. **Bulk errors are not typed (Must, because it affects the error contract).** The spec returns `bulkRequest.failedIndex` and, on partial writes, `bulkRequest.fragmentType` on 409, 422, 404, and 500. `core.Error` / `core.APIError` drop both fields. A caller cannot tell "nothing was created" from "part of the request was created". See §2.1a for the proposed concept.
2. **No partial-failure recovery in the streaming helpers (Should).** `createBulkSeq` stops at the first error. The spec gives a precise way to recover without duplicates: look up `fragmentType`, resend only the missing positions, and then remove the `c8y_BatchId_*` fragment. Add `BulkStreamOptions.RecoverPartial bool`, backed by a per-resource recovery function (§2.1a). This is the main reason to use the SDK over raw HTTP for bulk work.

### 2.1a Proposed concept: batch outcomes

The `bulkRequestError` schema is not specific to managed objects. The latest spec already uses it for three operations: `POST /inventory/managedObjects`, `PUT /inventory/managedObjects`, and `POST /measurement/measurements`. Other collection endpoints (child references, trusted-certificate bulk upload) are likely to adopt it. So the concept should live in `core` and `op`, not in `managedobjects`, and it should extend the two concepts callers already use: `core.Error` for errors and `op.Status` for outcomes.

**Layer 1: parse it generically in `core`.** Whenever an error body contains `bulkRequest`, the error becomes a `*core.BatchError`, whatever the endpoint:

```go
// package core

// BatchError is returned by any request that carries several items when the
// platform reports which item failed. Use errors.As to get it.
type BatchError struct {
    *Error                 // HTTP status, error type and message, as today

    // Index is the position of the failing item in the request, counted from 0.
    // It is -1 when the error is about the whole request (too many items, none).
    Index int

    // Applied says whether anything was written before the failure.
    Applied BatchApplied

    // Marker identifies what was written, so it can be found again.
    // For managed objects it is the c8y_BatchId_* fragment type.
    Marker string
}

type BatchApplied int

const (
    BatchAppliedNone    BatchApplied = iota // validated first, nothing written (most 4xx)
    BatchAppliedPartial                     // some items may have been written (Marker is set)
    BatchAppliedUnknown                     // the platform gave no information
)
```

`Applied` comes from the response: `fragmentType` present → `Partial`; a validation status (404, 409, or 422 without `fragmentType`) → `None`; otherwise `Unknown`. Callers can check this without knowing which endpoint they called:

```go
var be *core.BatchError
if errors.As(result.Err, &be) && be.Applied == core.BatchAppliedPartial {
    // part of the request was written
}
```

**Layer 2: one new outcome in `op`.** Add `StatusPartial Status = "Partial"` next to `StatusDuplicate`, `StatusSkipped`, and the other statuses. `Result.Status` is `StatusPartial` when `Applied == BatchAppliedPartial`, so code that already uses `switch result.Status` handles it. `IsError()` stays true.

**Layer 3: recovery per resource, with one shared shape.** Only the resource knows how to find what was written (managed objects: query by `fragmentType`; measurements: no marker today). Each resource that supports recovery exposes a function with the same shape, so `pipeline` and the stream helpers can use any of them:

```go
// package pipeline (or op)
type BatchRecoverer[T any] func(ctx context.Context, items []any, err *core.BatchError) ([]T, error)

// managedobjects implements it for bulk create, following the spec's steps:
// find by Marker, resend only the missing positions, then remove the marker.
func (s *Service) RecoverCreateBulk(ctx context.Context, items []any, err *core.BatchError) ([]jsonmodels.ManagedObject, error)
```

**Stream helpers:** `BatchError.Index` is relative to one request. When `CreateBulkStream` splits input into requests of 50, it must convert the index to the position in the whole input stream. Add `BatchError.Offset` (the position of the request's first item in the stream), so `Offset + Index` is the global position and `Index` keeps its meaning from the spec.

This keeps the existing contracts (`errors.As` on `core.Error`, `op.Status` switches). It needs one new error type and one new status, and no changes to the signatures of `CreateBulk`, `UpdateBulk`, or the measurement methods.
3. **No validation of the request size (Should).** `CreateBulk` / `UpdateBulk` accept any slice length. More than 50 items fails on the server with 422. Either reject it client-side with a clear error, or batch transparently the way the stream variants do.
4. **No typed `externalIds` on create (Should).** Add a small typed helper, for example `managedobjects.WithExternalIDs(body, identity...)`, or document `model.ManagedObject.ExternalIDs` in the `Create` godoc. Note that the spec rejects duplicate external IDs within one request.
5. **`GetOrCreateByExternalID` can be atomic (Should).** It currently does lookup → create → assign identity → delete on 409 (`service.go:509-560`). That leaves a window where a managed object exists without its external ID, and the compensating delete can fail. With `externalIds` on create, the flow becomes lookup → create with the ID → on 409, fetch the existing object. That is one write, with no compensation step. Keep the old path as a fallback for platforms that predate `externalIds` on create.
6. **Bulk update cannot change `externalIds` (doc only).** The spec says to use the Identity API. Add this to the `UpdateBulk` godoc.

---

## 3. Certificate login and enabling it for users

**What exists:** `LoginMethodCertificate` uses mTLS on port 8443 against `POST /devicecontrol/deviceAccessToken` (`client.go:1763`). That path is for **devices**. The automatic token refresh (`fetchToken`) uses the same path.

**What the new API adds:** `POST /tenant/oauth/certificate` works over standard HTTPS, with no mTLS and no port 8443. It covers regular users and microservice service users as well as devices, but **only for RSA keys**. The token comes back encrypted to the certificate's public key, and the platform only has a secure way to do that for RSA. EC certificates must keep using mTLS on port 8443. A platform bug currently lets EC certificates use the HTTPS exchange, but that is being changed, so the SDK must not rely on it.

- Request body: PEM leaf or chain (`application/x-pem-file`). The tenant comes from `?tenant_id=` or `X-Cumulocity-TenantId`. TFA is sent in `X-Cumulocity-TFA-Code` (the certificate replaces the password, not the second factor).
- Response: `{access_token, token_type, expires_in}`, where `access_token` is a **JWE** (`alg: RSA-OAEP-256`, `enc: A256GCM`) encrypted to the certificate's public key. The client must decrypt it with the private key to get the bearer JWT. Only RSA keys work. `CertificateKey` can hold either an RSA or an EC key, so the SDK has to choose the transport from the key type (see below).
- Opt-in: a regular user sets `requiredAuthType: CERTIFICATES` with `PUT /user/currentUser` (self-service only). A service user calls `PUT /application/currentApplication/authType`. An admin can only reset a service user to `NONE` (`PUT /application/applications/{id}/authType`). After the switch, password-derived tokens are rejected immediately.

**Recommended design**

- **Must (public API shape): one login method, with the transport chosen by key type.** Since the transport depends on the key, users should not have to pick it. Keep a single `LoginMethodCertificate` and add a transport setting to `AuthOptions`:

  ```go
  type CertificateTransport string

  const (
      CertificateTransportAuto  CertificateTransport = ""      // default: RSA → HTTPS, EC/Ed25519 → mTLS
      CertificateTransportHTTPS CertificateTransport = "HTTPS" // POST /tenant/oauth/certificate (RSA only)
      CertificateTransportMTLS  CertificateTransport = "MTLS"  // POST /devicecontrol/deviceAccessToken on MTLSPort
  )
  ```

  `Auto` reads the private key type. An RSA key uses the HTTPS exchange. Any other key uses mTLS on `MTLSPort` (default 8443). `Auto` never falls back from mTLS to HTTPS for an EC key: that would only work through the platform bug and would break when it is fixed. Forcing `HTTPS` with a non-RSA key fails locally with a clear error, before any request. The error for an mTLS failure should mention port 8443, because firewalls often block it. Fixing the name and the `Auto` rule now keeps the behavior of `CERTIFICATE` stable after v2.0.0.
- **Should:** `Tenants.LoginTokens.CreateWithCertificate(ctx, opts)` (or `authentication.CertificateTokenExchange`), plus JWE decryption. RSA-OAEP-256 + A256GCM is about 60 lines of standard library code (`crypto/rsa.DecryptOAEP` with SHA-256, then `crypto/cipher.NewGCM`), so it adds no new dependency. Wire both transports into `fetchToken` through the same `Auto` rule, so the cached token source refreshes the same way it does for password logins. Pass the TFA code through `LoginOptions`; the HTTPS exchange needs it for users with TFA.
- **Should:** typed opt-in helpers:
  - `jsonmodels.User.RequiredAuthType()` and `currentuser.Service.SetRequiredAuthType(ctx, "CERTIFICATES"|"NONE")`.
  - `currentapplication.Service.SetAuthType(ctx, …)` and `applications.Service.ResetAuthType(ctx, id)`.
  - Godoc warning on the "set" helpers about the lockout described in the spec. The SDK could also offer `EnableCertificateAuth(ctx, cert, key)`, which first proves that the exchange works and only then switches the account.
- **Could:** a `certutil` helper that builds the PEM chain from `tls.Certificate`. `certutil.CertificateChain` already does most of this for the `X-SSL-CERT-CHAIN` header.

---

## 4. Multi-tenant microservices (compared with the Java SDK)

**Feasibility: yes.** v2 already has the core building blocks, and they match the Java model:

| Java SDK | go-c8y v2 |
|---|---|
| Tenant scope / user scope | `TenantContext(TenantContextOptions{UseServiceUser})` middleware; `api.WithAuth` / `api.WithServiceUser` on `context.Context` |
| `runForEachTenant` | `ms.ForEachTenant(ctx, fn)` |
| `contextService.callWithinContext(creds, …)` | `ms.WithServiceUser(ctx, tenant)` |
| Per-tenant API beans | One shared `*api.Client`; credentials come from the context per request, with optional cached per-tenant OAI-Secure tokens (`ContextTokenExchange`) |
| Health endpoint | `RegisterHealthEndpoints(mux)` (`net/http`, no framework required) |

Because the client is shared and context-scoped, this is more efficient than the Java approach of per-tenant beans: one connection pool, no per-tenant goroutines, and cached tokens. That part of the design should stay.

**Gaps, in priority order**

1. **Security (Must): incoming credentials are never validated.** `authFromRequest` (`pkg/microservice/requestcontext.go:82`) splits the Basic header, or reads the JWT `ten` claim **without verifying the signature**. With `UseServiceUser: true`, it then runs the request as that tenant's service user. Anyone who can reach the container port with `Authorization: Basic base64("t123/x:y")` gets service-user access to tenant `t123`. The platform proxy normally authenticates first, but the Java SDK still verifies every request by calling `/user/currentUser` with the forwarded credentials.

   **Proposal: validation that can be turned off, with a cache so it costs little.** One extra platform call per request would be too expensive for busy microservices with many distinct users. So validation is a setting, and when it is on it is cached:

   ```go
   type CredentialValidation int

   const (
       // ValidateDefault: ValidateCached when UseServiceUser is true, ValidateNone otherwise.
       ValidateDefault CredentialValidation = iota
       ValidateNone   // trust the platform proxy; no extra calls (opt-out)
       ValidateCached // GET /user/currentUser once per credential, then cache
   )

   type TenantContextOptions struct {
       // ...existing fields
       Validation CredentialValidation
       ValidationTTL time.Duration                 // default 5m, never past the token's exp claim
       Validator     func(context.Context, authentication.AuthOptions) (CallerInfo, error) // optional custom check
   }
   ```

   Why the default differs by mode:
   - **User scope (default mode): no validation by default.** The downstream call already runs with the caller's own credentials, so the platform rejects bad credentials anyway. A separate check would only add a request. The one unverified value is the tenant from `TenantFromContext`; document that it is unverified in this mode.
   - **Service-user scope (`UseServiceUser: true`): cached validation by default.** Here the caller's credentials are never sent to the platform, so without a check anyone can act as the service user. Users who only expose the service through the platform proxy can still opt out with `ValidateNone`.

   How the cache keeps the load low:
   - Key: SHA-256 of the `Authorization` header or the authorization cookie, never the raw credential.
   - Each distinct credential costs **one** `/user/currentUser` call per TTL, however many requests it makes. Concurrent first requests share one call (`singleflight`), so a burst from one user makes one call.
   - The TTL is capped at the bearer token's `exp`, so a cached entry never outlives the token.
   - Failures are cached briefly (for example 10 s) so bad credentials can't hammer the platform. The cache has a size limit (LRU) so many distinct users can't grow memory without bound.
   - The validated response gives the tenant, user name, and `effectiveRoles`. These are stored in the request context (`CallerInfo`), so `TenantFromContext` returns the verified tenant and the role middleware in item 2 needs no extra call.
   - The `Validator` hook lets users plug in their own check, for example a shared cache across replicas.
2. **Role checks (Should).** There is no equivalent of `@PreAuthorize("hasRole(…)")`. Add `RequireRoles("ROLE_X", …)` middleware that uses `effectiveRoles` from the cached `CallerInfo` in item 1. It needs validation, so `RequireRoles` turns on `ValidateCached` for the routes it wraps, even when the rest of the service uses `ValidateNone`.
3. **Subscription lifecycle (Should).** Service users are loaded once at bootstrap and refreshed only on a cache miss (`GetServiceUser`). The Java SDK polls subscriptions (`C8Y.bootstrap.delay`) and fires `MicroserviceSubscriptionAddedEvent` / `RemovedEvent`. Add `ms.WatchSubscriptions(ctx, interval, OnTenantAdded, OnTenantRemoved)`, which diffs the user list and also detects changed passwords. Without it, per-tenant setup (Notification 2.0 subscriptions, caches) cannot react to unsubscribes. Unsubscribed tenants also stay in `ServiceUsers` until the next miss-triggered refresh.
4. **Bounded concurrency in `ForEachTenant` (Should).** It runs tenants one after another and stops at the first error. Add options for workers and continue-on-error. This needs no new dependency:
   - `github.com/destel/rill` (v0.9.0) is **already** a dependency. `pagination/executor.go` uses `rill.OrderedMap` to fetch pages in parallel. rill has no dependencies of its own (its `go.mod` has no `require` lines), so it adds nothing to the dependency tree.
   - The SDK's own `pipeline.ForEach` already provides workers, delay, `MaxErrors`, `ErrorThreshold`, and progress callbacks. `ForEachTenant` should be a thin wrapper over it, which keeps the options the same as the rest of the SDK:

     ```go
     func (m *Microservice) ForEachTenant(ctx context.Context, opts pipeline.Options,
         fn func(ctx context.Context, user model.ServiceUser) error) error {
         users := pipeline.FromSlice(m.serviceUsersSnapshot())
         return pipeline.ForEach(ctx, users, opts, func(ctx context.Context, u model.ServiceUser) error {
             return fn(api.WithServiceUser(ctx, u), u)
         })
     }
     ```

   **Risk of rill:** it is pre-1.0 and maintained by one person. That risk is acceptable as long as rill stays an **internal** detail. Today no exported SDK signature uses a rill type, and `pipeline` exposes only standard `iter.Seq2` and SDK-owned types. Keep it that way: if rill changes or is abandoned, only `executor.go` needs rewriting. (The `UpdateBulkStream` godoc suggests composing input with `rill.Batch`; point it to `pipeline.Batch` instead so users don't depend on rill through the SDK's docs.)
5. **Per-tenant settings (Should).** `Configuration` is a single viper instance. The Java SDK's `MicroserviceSettingsService` reads `GET /application/currentApplication/settings` **per tenant** (subscribers can override editable settings) with a 10-minute cache. Add `ms.Settings(ctx)`, which uses the tenant in `ctx`, caches the result, and handles `credentials.*` keys.
6. **Notification 2.0 per tenant (Should).** `SubscribeToNotifications` uses CometD realtime. Its reader goroutine never exits and never takes a context: `close(ch)` sits in the reader's own `defer`, which never runs. Provide a Notification 2.0 helper that is tenant-aware and context-cancelled (create the subscription and token as the tenant's service user, and tear it down on `OnTenantRemoved`), and fix or deprecate the CometD helper.
7. **Service-user certificate auth (Could, depends on §3).** A microservice can switch its service users to `CERTIFICATES`. Supporting this needs the §3 exchange plus a per-tenant certificate source in `ContextTokenExchange`. Service-user certificates should be RSA so they can use the HTTPS exchange. Microservice containers usually cannot rely on reaching port 8443.
8. **PER_TENANT isolation and `X-Cumulocity-Application-Key` (Could).** Detect `C8Y_MICROSERVICE_ISOLATION=PER_TENANT` and use `C8Y_TENANT/USER/PASSWORD` as the single service user. Also send the application key header that the platform expects on application-originated calls, if `APPLICATION_KEY` is set.

---

## 5. Design changes to make before freezing v2

These change exported API, so they are cheapest to fix now.

1. **The `resty` beta is part of the public API (Must).** `go.mod` pins `resty.dev/v3 v3.0.0-beta.6.<pseudo-version>`. Exported symbols use resty types, so any breaking change in resty becomes a breaking change in go-c8y v2.

   **Evidence that the risk is real:** the upgrade to `v3.0.0-rc.4` (now done on this branch) needed these changes between the pinned beta and rc.4:
   - Six renames, in `core`, `api`, and `tenants/logintokens`: `SetDoNotParseResponse` → `SetResponseDoNotParse`, `SetOutputFileName` → `SetResponseSaveFileName`, `SetSaveResponse` → `SetResponseSaveToFile`, `SetAllowNonIdempotentRetry` → `SetRetryAllowNonIdempotent`, the `ForceResponseContentType` field → `SetResponseForceContentType`, and `NewCircuitBreakerWithCount` → `NewCircuitBreakerCount`.
   - A **silent behavior change** the compiler could not catch. rc.4 clears the request body buffer and returns it to a pool as soon as the request finishes. Because `op.Result.Request` is resty's `RawRequest`, a dry-run request's body read as empty afterwards, and could even show another request's body once the buffer was reused. Three offline tests caught it. The fix copies the body in `DryRunTransport` (`snapshotRequestBody`), with a regression test.

   A user who writes a `PrepareRequest func(*resty.Request)` hook, or reads `RawRequest`, would hit the same kinds of breaks when upgrading go-c8y.

   **Where resty is visible** (counted with `go doc -all`): `pkg/c8y/api` (23 references), `pkg/c8y/api/core` (15), and `pkg/c8y` (1). So the work is contained:
   - `api`: `Client.HTTPClient *resty.Client`, `RequestOptions.PrepareRequest func(*resty.Request)`, `OnResponse func(*resty.Response)`, `RequestResult.Response *resty.Response`, and every `Middleware*` / `TokenRenewalRetry*` returning resty middleware types.
   - `core`: `Service.Client *resty.Client`, `TryRequest.Request`, `NewTryRequest`, `ExecuteResponseOnly`, `BinaryResponse.Response`, `NewMultiPartFile*`, `NewResult`.

   **Plan:**
   1. ~~Upgrade to `v3.0.0-rc.4`~~ Done: the SDK now tracks the release candidate rather than an old beta snapshot.
      - Remaining risk: `op.Result.Request` still exposes resty's `RawRequest` for real (non-dry-run) requests. After the call, its `GetBody` can return bytes from a reused pooled buffer. Either set `Result.Request` only in dry-run mode, as `docs/REQUEST_INSPECTION.md` already says it is, or give it an SDK-owned copy.
   2. Make the public contracts SDK-owned and based on the standard library, so later resty changes stay internal:
      - Middleware: `type Middleware func(next http.RoundTripper) http.RoundTripper`. Wrapping a `RoundTripper` is the standard way to extend Go HTTP clients, it works with any HTTP library, and users already know it. Convert the built-in `Middleware*` functions to it, or keep them unexported.
      - Hooks: `PrepareRequest func(*http.Request) error`, `OnResponse func(*http.Response) error`, `RequestResult.Response *http.Response`.
      - Multipart and binary: SDK types (`core.MultipartField`, `BinaryResponse` with `*http.Response`).
      - `core.Service.Client` and `Client.HTTPClient`: make them unexported, with one documented escape hatch, for example `func (c *Client) Resty() *resty.Client` (godoc: "not covered by the v2 compatibility guarantee"). Users who need resty-level control keep it, and everyone else is protected.
      - `core` is imported by the ~66 service packages, so rather than moving it to `internal`, stop embedding `core.Service` in the exported service types (hold it in an unexported field). Otherwise its fields are promoted into every service's public API.
   3. When resty v3.0.0 is released, upgrade. Only internal code should need changes.
2. **Two context-helper packages (Must).** `pkg/c8y/api/context` (processing mode only, and its name shadows the standard `context` package, so every caller needs an alias) and `pkg/c8y/api/contexthelpers` (processing mode, plus auth, dry run, deferred execution, redaction, and more) overlap. Merge them into one package and drop the package named `context`.
3. **Tenant repeated on every call (Should, breaking if changed later).** `users` and `usergroups` options have `Tenant string \`url:"-"\`` on each call, while other services resolve `{tenantId}` from the client or from `api.WithTenant(ctx)`. Make the field optional and default it the same way everywhere. Otherwise, document it as deliberate.
4. **Fragmented `applications` list API (Should).** `List`, `ListByName`, `ListByTenant`, `ListByOwner`, and `ListByUser` should be one `List(ListOptions{Name, Tenant, Owner, User, …})`, matching the other services. Do it now or keep them all permanently.
5. **Bulk error type (Must).** See §2.1. Error types are part of the contract.
6. **Login method naming (Must).** See §3.
7. **Bulk method signatures (Could).** `CreateBulk(ctx, []any)` returns `op.Result[jsonmodels.ManagedObject]`, the collection form. Consider a generic `[]T` parameter so typed slices (`[]model.ManagedObject`) work without converting to `[]any`.
8. **Leftover v1 artifacts (Must, documentation).** `README.md` still documents the v1 API (`github.com/reubenmiller/go-c8y/pkg/c8y`, `c8y.NewClientFromEnvironment(nil, false)`, "no guarantees on API stability"). `docs/V2.md` and `docs/DEFERRED_EXECUTION.md` import paths without `/v2`. `pkg/c8y/*.go` (6 v1 helper files) and `api/client_compat.go` (`SendRequest`, `RequestResult`) should either be documented as supported v2 API or moved to `internal` before tagging. Anything exported at v2.0.0 must be supported for the whole major version.
9. **Run `apidiff` / `gorelease` in CI after tagging (Could)**, so later releases cannot break the frozen API by accident.

---

## 6. Generic methods (Go 1.27)

**What I verified** by compiling small programs (Go 1.26.3 locally, Go 1.27.0 through toolchain download):

- Go 1.27 allows methods with their own type parameters, on both generic and non-generic types: `func (r Result[T]) Map[R any](fn func(T) R) Result[R]` and `func (d JSONDoc) Decode[T any]() (T, error)`. Go 1.26 rejects them ("method must have no type parameters").
- They are **promoted through embedding**. A generic method on `jsondoc.JSONDoc` is callable on every `jsonmodels` type that embeds it, for example `mo.Decode[MyDevice]()`. Method values work too.
- **Limits:**
  - Interfaces cannot declare generic methods ("interface method must have no type parameters").
  - A generic method does not satisfy a non-generic interface method of the same name.
  - A method cannot add a constraint to the receiver's own `T`. On `Result[T any]`, a method cannot require `T` to be `jsondoc.Unwrapper`, but it can constrain its own new type parameters.
- Cost: using them requires `go 1.27` in `go.mod`, so every user of the SDK needs Go 1.27 or newer. The SDK currently requires Go 1.26.

**Where they would improve the API.** Many of v2's generic helpers are free functions only because methods could not take type parameters. Moving them onto the types makes them discoverable with autocomplete and allows chaining:

| Today (free function) | With generic methods | Notes |
|---|---|---|
| `op.MapResult(r, fn)` | `r.Map(fn)` | Direct fit |
| `op.FlatMap(r, fn)` | `r.FlatMap(fn)` | Direct fit |
| `op.IterAs[U](r)`, `op.IterAsErr[U](r)` | `r.IterAs[U]()` | `T Unwrapper` must become a runtime check (see limits) |
| `jsondoc.Decode[T](d)` | `d.Decode[T]()`, so `mo.Decode[MyDevice]()` on every model | Direct fit, through embedding |
| `jsonmodels.GetAs[T](mo, key)`, `GetFragment[T](mo)` | `mo.GetAs[T](key)`, `mo.Fragment[T]()` | Method-level constraint `T model.Fragment` is allowed |
| `pagination.Iterator[T]` plus `op.Iter*` helpers | `it.Map(fn)`, `it.As[U]()` | Lets `ListAll(...)` chain into typed values |
| `op.Iter`, `op.Iter2`, `op.ToSliceR`, `op.CollectR` | Already covered by `r.Items()`; add `r.Collect()` | Remove the duplicate functions |

**Keep as functions:**
- Constructors: `op.OK[T]`, `pagination.Paginate[T, D]`.
- Functions over several values: `op.Combine`.
- The `pipeline` step combinators (`Pipe`, `Retry`, …): they compose functions, not methods.

**Design rule to adopt:** service methods (`Get`, `List`, `Create`, …) stay non-generic, so users can still wrap services in their own interfaces for mocking. Generic methods should be **convenience methods on value types only**: `op.Result`, `pagination.Iterator`, `jsondoc.JSONDoc`. Do not add `Service.GetAs[T]()`-style methods. They can't be part of an interface, and `client.ManagedObjects.Get(ctx, id, opts).Decode[MyDevice]()` gives the same result while keeping services mockable.

**Recommendation (Should, decide before tagging):** adding methods later is backward compatible, so this is not strictly a v2.0.0 blocker. But if you adopt them before tagging, you can **remove** the free-function duplicates and ship one way of doing each thing. After v2.0.0, both forms must stay for the whole major version. The trade-off is requiring Go 1.27. Go 1.26 remains a supported Go release until Go 1.28 ships, so some users may not have upgraded yet. If you want v2.0.0 soon, tag on Go 1.26 and add the methods in v2.x alongside the existing functions.

---

## Suggested order

1. Must items that change exported API: §5.1 (resty rc.4 upgrade, then standard-library contracts), §5.2, the batch outcome concept (§2.1a / §5.5), the certificate transport rule (§3 / §5.6), §5.8. Decide on Go 1.27 generic methods (§6) at the same time, because adopting them before tagging lets you remove the duplicate free functions.
2. Security fix: §4.1, validating incoming credentials (cached, with an opt-out).
3. Vendor the new spec and add the missing operations in §1. Add the certificate exchange and opt-in helpers from §3.
4. Microservice lifecycle: §4.3 `WatchSubscriptions`, §4.5 settings, §4.6 Notification 2.0, §4.2 roles.
5. Bulk recovery helpers and atomic `GetOrCreateByExternalID` (§2.2–2.5).
6. Tag `v2.0.0`. Everything after step 2 is additive and can also ship in v2.x.
