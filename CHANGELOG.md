# CHANGELOG

## Unreleased

* Added `Microservice.WatchSubscriptions` (tenant added, removed and credentials-changed callbacks), `Microservice.Settings` (per-tenant settings with a 10-minute cache, `SettingsTTL`), and `Microservice.SubscribeNotifications` (Notification 2.0 per tenant with acknowledgements, reconnect, and clean shutdown).
* `Microservice.ForEachTenant` accepts optional `pipeline.Options` to process tenants concurrently; without options it behaves as before.
* `PER_TENANT` isolation (`C8Y_MICROSERVICE_ISOLATION`) uses `C8Y_TENANT`/`C8Y_USER`/`C8Y_PASSWORD`; `APPLICATION_KEY` adds `X-Cumulocity-Application-Key`; a `401` on a service-user request refreshes the service users.
* Fixed `Notification2.SubscribeStream`, which never delivered messages (the registered channel was never read), and fixed `notification2.Notification2Client` goroutine leaks: `Close` now stops the hub, the writer, the worker and reconnect attempts.
* Fixed the reader goroutine leak in `Microservice.SubscribeToNotifications`; it and `SubscribeToOperations` are deprecated in favour of `SubscribeNotifications`.
* Vendored the latest core OpenAPI spec. Added `Tenants.RestrictedRoles`, `SmartRestTemplates.BindExternalID`, `TrustedCertificates.CreateBulk`, `UnassignMany` on `ChildDevices`/`ChildAssets`/`ChildAdditions`, `TOTP.GetActivity`, and `Applications.DownloadBinary`.
* Fixed `Applications.ListByOwner`, which called `/application/applicationsByTenant/{tenantId}` instead of `/application/applicationsByOwner/{tenantId}`.
* `c8ygen lint` now compares operations (method + path) and requires a reason for every drift waiver; waivers can name a method (`POST /path`).
* **Security, breaking (behavior)**: `Microservice.TenantContext` with `UseServiceUser: true` now validates the caller's credentials with `GET /user/currentUser` before acting as the tenant's service user (cached per credential). Previously any request naming a subscribed tenant got service-user access. Set `TenantContextOptions{Validation: microservice.ValidateNone}` to trust the platform proxy instead. New: `ValidationTTL`, `Validator`, `CallerInfo`, `CallerFromContext`, `RequireRoles`.
* Added certificate login over HTTPS (`POST /tenant/oauth/certificate`) for RSA keys, including JWE token decryption, TFA, and token refresh. `LoginMethodCertificate` now picks the transport from the key type through `AuthOptions.CertificateTransport` (`Auto`, `HTTPS`, `MTLS`): RSA keys use HTTPS, EC and Ed25519 keys keep using mTLS on port 8443, with no fallback between them. **Breaking (behavior)**: RSA client certificates now use the HTTPS exchange instead of mTLS by default; set `CertificateTransportMTLS` to keep the previous behavior.
* Added certificate opt-in helpers: `CurrentUser.SetRequiredAuthType`, `Applications.Current.SetAuthType`, `Applications.ResetAuthType`, `User.RequiredAuthType()`, and `Client.EnableCertificateAuth`, which checks the certificate before opting in.
* Added `TrustedCertificates.VerifyChain`, `LoginTokens.CreateWithCertificate`, and `ClientOptions.ContextCertificateSource` for certificate-based service users.
* Added batch outcomes for requests that carry several items: `core.BatchError` (`Index`, `Offset`, `Applied`, `Marker`), produced for any error with a `bulkRequest` body (managed object bulk create/update, `Measurements.CreateList`), and the result status `op.StatusPartial` when part of a request was written. Batch failures are never marked retryable.
* Added `ManagedObjects.RecoverCreateBulk` (signature `pipeline.BatchRecoverer`) and `BulkStreamOptions.RecoverPartial` to complete a partially written bulk create without duplicates.
* Added `managedobjects.ExternalID` and `managedobjects.WithExternalIDs` to bind external IDs on create.
* `CreateBulk` and `UpdateBulk` reject empty requests, more than 50 items, and (create) an external ID used twice, before sending anything.
* `GetOrCreateByExternalID` now creates the managed object and its external ID in one request. A conflict returns the existing object without deleting anything. Older platforms fall back to the previous create-then-assign flow.
* **Breaking**: resty types are no longer part of the public API, so resty upgrades cannot break go-c8y users. Migration:

  | Before | After |
  |---|---|
  | `client.HTTPClient` | `client.Resty()` (escape hatch, not covered by the v2 compatibility guarantee) |
  | `client.HTTPClient.AddResponseMiddleware(api.MiddlewareCountByMethodAndPath(stats))` | `client.Use(api.CountByMethodAndPath(stats))` or `ClientOptions.Middleware` |
  | `api.MiddlewareCountRequests(&n)` | `api.CountRequests(&n)` |
  | Custom resty middleware | `api.Middleware` (`func(next http.RoundTripper) http.RoundTripper`), via `ClientOptions.Middleware` or `client.Use` |
  | `RequestOptions.PrepareRequest func(*resty.Request)` | `RequestOptions.PrepareRequest func(*http.Request) error` |
  | `RequestOptions.OnResponse func(*resty.Response) error` | `RequestOptions.OnResponse func(*http.Response) error` |
  | `RequestResult.Response *resty.Response` | `RequestResult.Response *http.Response` (its `Request` is an SDK-owned copy) |
  | `core.BinaryResponse.Response *resty.Response` | `*http.Response`; new `BinaryResponse.StatusCode()` |
  | `api.MiddlewareAddUserAgent`, `MiddlewareAddHost`, `MiddlewareRemoveEmptyTenantID`, `MiddlewareContextAuthorization`, `TokenSourceMiddleware`, `TokenRenewalRetry`, `SetAuth`, `NoAcceptHeader` | Internal; `NewClient` installs them. Set the user agent with `ClientOptions.Agent` |
  | `api.MiddlewareAddCookies`, `MiddlewareAuthorization`, `TokenRenewalRetryMiddleware`, `api.Service`, `api.ExecuteNoContent`, `api.ExecuteResponseOnly`, `api.UnmarshalJSON` | Removed (unused) |
  | `core.Service`, `core.TryRequest`, `core.NewTryRequest`, `core.Execute*`, `core.QueryParameters`, `core.NewMultiPartFile*`, `core.NewBinaryResponse`, `core.Result`, `core.Response` | Internal (`pkg/c8y/internal/httpcore`) |
  | `Client.AddMiddleware()` | Internal; it ran once from `NewClient` |
* **Breaking**: services no longer embed the HTTP client, so `client.Alarms.Client` and similar fields are gone.
* Added `api.Middleware`, `api.RoundTripperFunc`, `ClientOptions.Middleware`, `Client.Use`, and `task check-api-surface` (run in CI) to keep resty types out of the public API.
* **Breaking**: request-scoped options now have one public home, package `api`. `pkg/c8y/api/context` is removed and `pkg/c8y/api/contexthelpers` moved to `pkg/c8y/internal/contexthelpers`. New in `api`: `WithProcessingMode` (plus `Persistent`, `Transient`, `Quiescent`, `CEP` variants), `ProcessingMode`, `WithExtraHeaders`, `ExtraHeaders`, `WithExtraQueryParams`, `ExtraQueryParams`, `WithNoAccept`, `WithNoAuth`, `WithDryRunHandler`, and `TenantFromContext`. Migration:

  | Before | After |
  |---|---|
  | `c8yctx.WithProcessingMode…(ctx)` (`api/context`) | `api.WithProcessingMode…(ctx)` |
  | `contexthelpers.With…(ctx, …)` | `api.With…(ctx, …)` |
  | `contexthelpers.GetProcessingMode(ctx)` | `api.ProcessingMode(ctx)` |
  | `contexthelpers.GetExtraHeaders(ctx)` / `GetExtraQueryParams(ctx)` | `api.ExtraHeaders(ctx)` / `api.ExtraQueryParams(ctx)` |
* **Breaking**: the tenant path parameter is no longer read from the untyped context key `"tenant"`. Use `api.WithTenant(ctx, tenant)`.
* Added context-scoped credentials to the core API client: `api.WithAuth`, `api.WithServiceUser` and `api.WithTenant` allow any request made with a shared client to execute on behalf of a specific tenant (see `docs/MICROSERVICE_TENANT_CONTEXT.md`)
* Added `Microservice.TenantContext()`, a framework-agnostic `net/http` middleware that binds the caller's tenant credentials to the request context (parity with the Java SDK's per-request security context), with optional service-user (tenant) scope via `TenantContextOptions{UseServiceUser: true}`
* Added `Microservice.ForEachTenant()` and `Microservice.WithServiceUser()` helpers for running code within a tenant's context (parity with the Java SDK's `runForEachTenant`/`runForTenant`)
* Added `microservice.New()` + `Microservice.Bootstrap(ctx)` for explicit, error-returning startup; `NewDefaultMicroservice` is deprecated but unchanged
* **Breaking**: `pkg/microservice` no longer depends on the echo web framework. The health endpoints are now plain `net/http` handlers; use `RegisterHealthEndpoints(mux)` or mount the handlers individually (echo users can wrap them with `echo.WrapHandler`)
* Added optional per-tenant token exchange: `api.ClientOptions{ContextTokenExchange: true}` (or `client.SetContextTokenExchange(true)`) exchanges per-request basic credentials for cached OAI-Secure bearer tokens, with automatic refresh, 401 invalidation and a basic-auth fallback when the tenant does not support OAI-Secure
* **Breaking**: `pkg/microservice` no longer depends on the cron library; the `Scheduler` type, `StartOperationPolling` and the `agent.operations.pollRate` property have been removed. The SDK runs no background timers of its own — call `CheckForNewConfiguration()` from your own `time.Ticker` (or a realtime subscription) to react to pending operations
* Added `authentication.FromLookup`, `authentication.HostFromLookup`, `authentication.LookupFunc` and `authentication.LookupFromMap` to resolve the `C8Y_*` settings from any key/value source (e.g. a credentials file) with the same rules as `FromEnvironment`/`HostFromEnvironment`, which now delegate to them (behaviour unchanged)
* Added `Client.AuthorizationHeader(ctx)` returning the `Authorization` header value the client would send for a request with that context (context credentials, token source with renewal, then the client credentials as the HTTP client sends them: a token wins over Basic), for requests forwarded without going through the client. `MiddlewareAuthorization` no longer logs the bearer token
* Added `ClientOptions.SkipTenantLookup` to skip the `/tenant/currentTenant` call `NewClient` makes when no tenant is given (for clients built only to produce credentials or headers), and `Client.ResolveTenant(ctx)` to run that lookup later with the caller's context

## Released

### v0.14.0

This release is a cleanup of the project which merges a long standing parallel branch. The parallel branch included a lot of changes which were used by the [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) project. And like with any long lasting branches, it was hard to list all of the changes that were done since the last official release. Moving forward all releases will go through a more formal release process.

* Dry option improvements
    * Removed unnecessary indentation when displaying body in prettified json
    * Added dry output of form data information for PUT and POST requests
    * Dry run now displays `Body: (empty)` for PUT and POST requests when the input body is set to `nil`

* Added support for non-json type bodies

* Write log output `Body: (empty)` in the dry run if the request method is not PUT, PATCH or POST even if a body is provided

* Prevent nil panic by checking for an error when creating a new request

* Hide OAuth2 authorization cookie value and Xsrf Token when hide sensitive information is enabled
* Hide Host path when hide sensitive information is enabled
* Removed `EnforceStrength` in login options as it has changed from string to bool which was causing a parsing error.
* Added common request options (only supports DryRun for now)
* Added DryRunResponse option to return a fake response containing the Request that would have been sent
* Added `UnsilenceLogger` to re-enable logger output after using `SilenceLogger`
* Removed newline endings in log messages
* Fixed invalid options for `GetNewDeviceRequests`
* Added additional properties (owner, tenantId, creationTime) to `NewDeviceRequest`
* Added support for using bearer authorization
* Fixed bug when hiding tokens when it is empty

### v0.8.0

* Migrated to using github actions to run integration tests
* Added integration tests against a real tenant
* Fixed linting
* Fixed bug when uploading microservice binary where the `GET` method was being used instead of `POST`
* Added VS Code dev container to make it easier to contribute to project
