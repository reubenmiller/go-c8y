# CHANGELOG

## Unreleased

v2 is a new module, `github.com/reubenmiller/go-c8y/v2`. Entries marked **Breaking** change behavior or remove API compared with earlier v2 pre-releases; each one says what to use instead.

### Go version, results and JSON

* **Breaking**: go-c8y v2 requires Go 1.27. Generic helpers are now methods (Go 1.27 generic methods), and the free functions they replace are removed:

  | Before | After |
  |---|---|
  | `op.MapResult(r, fn)` / `op.FlatMap(r, fn)` | `r.Map(fn)` / `r.FlatMap(fn)` |
  | `op.Iter2(r)` | `r.Items()` |
  | `op.Iter(r)` (panicked on errors) | `r.Seq()` (stops at the first error) |
  | `op.ToSliceR(r)` / `op.CollectR(r)` | `r.Collect()` |
  | `op.IterAs[U](r)` / `op.IterAsErr[U](r)` | `r.IterAs[U]()` (yields errors) |
  | `jsondoc.Decode[T](d)` (returned `*T`) | `d.Decode[T]()` (returns `T`) |
  | `jsonmodels.GetAs[T](mo, key)` | `mo.GetAs[T](key)` |
  | `jsonmodels.GetFragment[T](mo)` | `mo.Fragment[T]()` |
* **Breaking**: building an `op.Result` is now done with package functions instead of methods. Each method of a generic type is compiled for every `Result` instantiation (about 100) in every package that imports the SDK. With 30 methods, importing `api` alone compiled 5,291 functions into each importing package; it is now 3,615 (a cold go-c8y-cli build uses about 20% less CPU). The rule is: read a result with a method, build one with a function. Package functions are only compiled where they are called:

  | Before | After |
  |---|---|
  | `r.WithMeta(k, v)`, `r.WithHTTPStatus(s)`, `r.WithRequest(req)`, `r.WithResponse(b)`, `r.WithRequestID(id)`, `r.WithAttempts(n)`, `r.WithDuration(d)`, `r.WithExecutor(fn)` | `op.WithMeta(r, k, v)`, `op.WithHTTPStatus(r, s)`, ... (same name, the result first) |
  | `r.ExecuteOrDefer(ctx)` | `op.ExecuteOrDefer(ctx, r)` |
  | `r.TotalElements()`, `r.TotalPages()`, `r.CurrentPage()`, `r.PageSize()`, `r.HasNextPage()` | `r.Meta.TotalElements()`, ... (`Result.Meta` is now the named type `op.Meta`, still a `map[string]any`) |

  Everything that reads a result stays a method: `IsError`, `IsSuccess`, `IsNotFound`, `IsRetryable`, `IsDeferred`, `IgnoreNotFound`, `IgnoreConflict`, `IgnoreStatusCodes`, `Unwrap`, `Items`, `Seq`, `First`, `Collect`, `Execute`, `Documents` and the generic methods. A test in `pkg/c8y/internal/compilecost` fails when the count grows past its budget.
* Added `r.Decode[U]()` on results, and `Map` and `As[U]()` on `pagination.Iterator`. `model.ErrFragmentNotFound` is now the same value as `jsondoc.ErrKeyNotFound`.

### HTTP client and middleware

* Added `ClientOptions.Logger` (`*slog.Logger`, default `slog.Default()`). The HTTP layer's messages go to it instead of always to stderr: the request and response logs of `Debug` and its warnings (e.g. "Using sensitive credentials in HTTP mode") at info level, its errors at error level. An application can hide those warnings with a warn-level logger.
* **Breaking (behavior)**: deleting a resource that does not exist now succeeds. The result has no error, `Status` is `op.StatusSkipped`, `HTTPStatus` is `404`, and `Meta["ignoredStatus"]` is `404`. Most deletes already called `IgnoreNotFound()` internally (44 calls in 38 packages), but a delete's own `404` was never stored in `HTTPStatus`, so the error was always returned. Other delete errors (`401`, `403`, `409`, `5xx`) are still returned. To treat a missing resource as an error, check `result.Status == op.StatusSkipped`. Deletes without an internal `IgnoreNotFound()` now report `HTTPStatus`, so `IsNotFound()` and `IgnoreNotFound()` work on them.
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
* **Breaking**: the root package `pkg/c8y` now holds only the HTTP response cache (`NewCachedTransport`, `NewCachedClient`, `CacheOptions`) and `IsID`, all supported v2 API. The unused v1 leftovers are removed: `Response`, `ProxyReader`, `NewProxyReader`, `NewStringReader`, `NewByteReader`, `Upload`, `RequestMiddleware`, and `DecodeJSONBytes` (use `api.DecodeJSONBytes`). `IsID("")` now returns `false`. `pkg/c8y/api/mock` and `pkg/jsonUtilities` are now internal. `pkg/zipUtilities` stays public as supported v2 API.
* `Client.SendRequest`, `RequestOptions`, and `RequestResult` are supported v2 API for endpoints without a typed service method.
* Added `Client.AuthorizationHeader(ctx)` returning the `Authorization` header value the client would send for a request with that context (context credentials, token source with renewal, then the client credentials as the HTTP client sends them: a token wins over Basic), for requests forwarded without going through the client. `MiddlewareAuthorization` no longer logs the bearer token
* Added `ClientOptions.SkipTenantLookup` to skip the `/tenant/currentTenant` call `NewClient` makes when no tenant is given (for clients built only to produce credentials or headers), and `Client.ResolveTenant(ctx)` to run that lookup later with the caller's context

### Request context

* **Breaking**: request-scoped options now have one public home, package `api`. `pkg/c8y/api/context` is removed and `pkg/c8y/api/contexthelpers` moved to `pkg/c8y/internal/contexthelpers`. New in `api`: `WithProcessingMode` (plus `Persistent`, `Transient`, `Quiescent`, `CEP` variants), `ProcessingMode`, `WithExtraHeaders`, `ExtraHeaders`, `WithExtraQueryParams`, `ExtraQueryParams`, `WithNoAccept`, `WithNoAuth`, `WithDryRunHandler`, and `TenantFromContext`. Migration:

  | Before | After |
  |---|---|
  | `c8yctx.WithProcessingMode…(ctx)` (`api/context`) | `api.WithProcessingMode…(ctx)` |
  | `contexthelpers.With…(ctx, …)` | `api.With…(ctx, …)` |
  | `contexthelpers.GetProcessingMode(ctx)` | `api.ProcessingMode(ctx)` |
  | `contexthelpers.GetExtraHeaders(ctx)` / `GetExtraQueryParams(ctx)` | `api.ExtraHeaders(ctx)` / `api.ExtraQueryParams(ctx)` |
* **Breaking**: the tenant path parameter is no longer read from the untyped context key `"tenant"`. Use `api.WithTenant(ctx, tenant)`.
* Added context-scoped credentials to the core API client: `api.WithAuth`, `api.WithServiceUser` and `api.WithTenant` allow any request made with a shared client to execute on behalf of a specific tenant (see `docs/MICROSERVICE_TENANT_CONTEXT.md`)

### Authentication and certificates

* Added certificate login over HTTPS (`POST /tenant/oauth/certificate`) for RSA keys, including JWE token decryption, TFA, and token refresh. `LoginMethodCertificate` now picks the transport from the key type through `AuthOptions.CertificateTransport` (`Auto`, `HTTPS`, `MTLS`): RSA keys use HTTPS, EC and Ed25519 keys keep using mTLS on port 8443, with no fallback between them. **Breaking (behavior)**: RSA client certificates now use the HTTPS exchange instead of mTLS by default; set `CertificateTransportMTLS` to keep the previous behavior.
* Added certificate opt-in helpers: `CurrentUser.SetRequiredAuthType`, `Applications.Current.SetAuthType`, `Applications.ResetAuthType`, `User.RequiredAuthType()`, and `Client.EnableCertificateAuth`, which checks the certificate before opting in.
* Added `TrustedCertificates.VerifyChain`, `LoginTokens.CreateWithCertificate`, and `ClientOptions.ContextCertificateSource` for certificate-based service users.
* Added `TrustedCertificates.CertificateAuthority.Update` and the `CreateOptions` settings `Status` and `AutoRegistrationEnabled`, which `Create` applies when they differ. On a 409 (already exists), `Create` now returns the existing certificate authority instead of an empty result.
* **Breaking (behavior)**: in a dry run, `CertificateAuthority.Delete` and `Update` send the lookup of the CA certificate and return the prepared request with the real fingerprint. Before, the lookup was also a dry run, found nothing, and `Delete` reported "nothing to delete".
* Fixed two errors returned as `core.Error` values instead of `*core.Error` (no certificate authority found, and a 502 HTML page from a proxy), which `core.ErrHasStatus` and `errors.As(err, &*core.Error)` did not recognize. The certificate authority one now has the message "no certificate authority found".
* Added optional per-tenant token exchange: `api.ClientOptions{ContextTokenExchange: true}` (or `client.SetContextTokenExchange(true)`) exchanges per-request basic credentials for cached OAI-Secure bearer tokens, with automatic refresh, 401 invalidation and a basic-auth fallback when the tenant does not support OAI-Secure
* Added `authentication.FromLookup`, `authentication.HostFromLookup`, `authentication.LookupFunc` and `authentication.LookupFromMap` to resolve the `C8Y_*` settings from any key/value source (e.g. a credentials file) with the same rules as `FromEnvironment`/`HostFromEnvironment`, which now delegate to them (behaviour unchanged)

### Bulk requests and batch outcomes

* Added batch outcomes for requests that carry several items: `core.BatchError` (`Index`, `Offset`, `Applied`, `Marker`), produced for any error with a `bulkRequest` body (managed object bulk create/update, `Measurements.CreateList`), and the result status `op.StatusPartial` when part of a request was written. Batch failures are never marked retryable.
* Added `ManagedObjects.RecoverCreateBulk` (signature `pipeline.BatchRecoverer`) and `BulkStreamOptions.RecoverPartial` to complete a partially written bulk create without duplicates.
* Added `managedobjects.ExternalID` and `managedobjects.WithExternalIDs` to bind external IDs on create.
* `CreateBulk` and `UpdateBulk` reject empty requests, more than 50 items, and (create) an external ID used twice, before sending anything.
* `GetOrCreateByExternalID` now creates the managed object and its external ID in one request. A conflict returns the existing object without deleting anything. Older platforms fall back to the previous create-then-assign flow.

### Microservices

* `op.Result.Documents` yields one document per item when the result is a collection (its data is a JSON array), like a page of a pagination iterator. Before, the whole array was one document.
* Added `Microservices.ListServiceUsers`, which lists the service users of a microservice (one per subscribed tenant): it resolves the microservice, fetches its bootstrap user and reads the subscriptions with the bootstrap user's credentials. The fake server now serves `GET /application/applications/{id}/bootstrapUser`.
* **Security, breaking (behavior)**: `Microservice.TenantContext` with `UseServiceUser: true` now validates the caller's credentials with `GET /user/currentUser` before acting as the tenant's service user (cached per credential). Previously any request naming a subscribed tenant got service-user access. Set `TenantContextOptions{Validation: microservice.ValidateNone}` to trust the platform proxy instead. New: `ValidationTTL`, `Validator`, `CallerInfo`, `CallerFromContext`, `RequireRoles`.
* Added `Microservice.TenantContext()`, a framework-agnostic `net/http` middleware that binds the caller's tenant credentials to the request context (parity with the Java SDK's per-request security context), with optional service-user (tenant) scope via `TenantContextOptions{UseServiceUser: true}`
* Added `Microservice.ForEachTenant()` and `Microservice.WithServiceUser()` helpers for running code within a tenant's context (parity with the Java SDK's `runForEachTenant`/`runForTenant`)
* Added `microservice.New()` + `Microservice.Bootstrap(ctx)` for explicit, error-returning startup; `NewDefaultMicroservice` is deprecated but unchanged
* **Breaking**: `pkg/microservice` no longer depends on the echo web framework. The health endpoints are now plain `net/http` handlers; use `RegisterHealthEndpoints(mux)` or mount the handlers individually (echo users can wrap them with `echo.WrapHandler`)
* **Breaking**: `pkg/microservice` no longer depends on the cron library; the `Scheduler` type, `StartOperationPolling` and the `agent.operations.pollRate` property have been removed. The SDK runs no background timers of its own — call `CheckForNewConfiguration()` from your own `time.Ticker` (or a realtime subscription) to react to pending operations
* Added `Microservice.WatchSubscriptions` (tenant added, removed and credentials-changed callbacks), `Microservice.Settings` (per-tenant settings with a 10-minute cache, `SettingsTTL`), and `Microservice.SubscribeNotifications` (Notification 2.0 per tenant with acknowledgements, reconnect, and clean shutdown).
* `Microservice.ForEachTenant` accepts optional `pipeline.Options` to process tenants concurrently; without options it behaves as before.
* `PER_TENANT` isolation (`C8Y_MICROSERVICE_ISOLATION`) uses `C8Y_TENANT`/`C8Y_USER`/`C8Y_PASSWORD`; `APPLICATION_KEY` adds `X-Cumulocity-Application-Key`; a `401` on a service-user request refreshes the service users.
* Fixed the reader goroutine leak in `Microservice.SubscribeToNotifications`; it and `SubscribeToOperations` are deprecated in favour of `SubscribeNotifications`.

### Notification 2.0

* Fixed `Notification2.SubscribeStream`, which never delivered messages (the registered channel was never read), and fixed `notification2.Notification2Client` goroutine leaks: `Close` now stops the hub, the writer, the worker and reconnect attempts.

### API coverage

* Vendored the latest core OpenAPI spec. Added `Tenants.RestrictedRoles`, `SmartRestTemplates.BindExternalID`, `TrustedCertificates.CreateBulk`, `UnassignMany` on `ChildDevices`/`ChildAssets`/`ChildAdditions`, `TOTP.GetActivity`, and `Applications.DownloadBinary`.
* Added `Applications.DeployHosted`, which creates or updates a hosted web application from a folder (zipped with `pkg/zipUtilities`) or a `.zip` file, uploads the binary, and activates it. The name, key, and context path default to the `cumulocity.json` manifest, then to the file or folder name. `Owner` limits the lookup to applications owned by one tenant, `SkipUpload` and `SkipActivation` skip those steps, and `KeepBinaries` deletes the oldest binaries beyond that number (never the active or newly uploaded one). Also added `applications.ReadHostedManifest`. The fake server now stores application binaries (list, download, and delete) and filters applications by `owner`.
* Fixed `zipUtilities.ZipDirectoryContents`, `ZipDirectory`, and `ZipFile`: zip entry names now always use forward slashes (previously backslashes on Windows) and never start with "/", and errors from reading the source are returned instead of being ignored.
* Fixed `Applications.ListByOwner`, which called `/application/applicationsByTenant/{tenantId}` instead of `/application/applicationsByOwner/{tenantId}`.
* Fixed `zipUtilities.UnzipFile`, which extracted every entry except the given `names`; it now extracts only the named entries (case-insensitive), and everything when `names` is empty.

### Tooling and documentation

* `c8ygen lint` now compares operations (method + path) and requires a reason for every drift waiver; waivers can name a method (`POST /path`).
* The README describes v2 (install, quick start, Go 1.27, stability promise), and the docs use `/v2` import paths. `docs/API_DESIGN.md` explains the tenant default for tenant-scoped endpoints and how `Applications.List` differs from the `ListBy*` methods.
* CI checks the docs (`task check-docs`: no v1 import paths, every `package main` example builds) and runs `gorelease` against the latest v2 tag (`task check-api-compat`, blocking once `v2.0.0` exists).

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
