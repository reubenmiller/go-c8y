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
* `SendRequest` now handles a request like the typed service methods do:
  * **Breaking (behavior)**: a response with an HTTP error status sets `RequestResult.Error` to a `*core.Error` with the status, the Cumulocity error type and message (or the body, when it is not a Cumulocity error). The body stays readable with `Body()`. Before, `Error` was only set when the request could not be sent; `IsError()` is unchanged.
  * It applies the context's processing mode and custom headers and query parameters (`WithProcessingMode`, `WithExtraHeaders`, `WithExtraQueryParams`). The request's own options win.
  * A query parameter given several times (in the path or in `Query`) keeps every value. Before, only the last one was sent. `WithExtraQueryParams` now also keeps every value of a key for the typed service methods.
  * A `DELETE` request sends its `Body`. Before, the HTTP client dropped it.
  * With `Host`, the path's query parameters are sent once. Before, they were sent twice.
* Added `api.FormFile` and `api.FormField` to set how a `RequestOptions.FormData` value is sent: as a file part with a file name and content type, or as a plain form field. An `*os.File` is now sent as a file part named after the file, with a content type detected from the file name or the contents (before: named after the field, as `application/octet-stream`). Other readers are sent as before. The parts are sent in field name order.
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

* Added a public request hook and transfer progress, so SDK users (and go-c8y-cli) can wrap request and response bodies without `SendRequest`:
  * `api.WithRequestHook(ctx, func(*http.Request) error)` runs on the final request of every call made with the context, typed or raw (also for dry runs), and may replace `req.Body`.
  * `api.WithResponseBodyWrapper(ctx, func(*http.Response) io.ReadCloser)` replaces the response body, for example for downloads.
  * `api.WithUploadProgress` and `api.WithDownloadProgress` report `(read, total)` through the new `core.ProgressFunc`; `core.NewProgressReader` and `core.ReaderSize` are the building blocks.
  * `core.UploadFileOptions.Progress` reports the uploaded file bytes of every typed upload (binaries, event binaries, applications and versions, microservices, firmware, software, configuration, certificate revocation lists, bulk device registration), with the file size as total.
* Added `Client.Request(ctx, RequestOptions) op.Result[jsondoc.JSONDoc]`, a typed escape hatch for endpoints without a service method. It shares middleware, authentication, dry run and error handling with `SendRequest` and the typed services.

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

* **Breaking**: `Microservices.CreateOrUpdate(ctx, opt) (*model.Microservice, error)` is now `Microservices.Deploy(ctx, opt) op.Result[jsonmodels.Microservice]`, like `Applications.DeployHosted`. The file is checked before anything is created, the lookup is sent also in a dry run (the later steps are skipped), the result's `Status` is `StatusCreated` or `StatusUpdated`, and deferred execution is supported.
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

### Websocket clients

The realtime, notification2 and remote access clients now connect like an HTTP call of the same `api.Client`.

* Added `Client.WebsocketDialer() *websocket.Dialer` (the TLS configuration of the client's transport, including `InsecureSkipVerify`, extra root CAs and client certificates, plus its proxy and dial function), `Client.WebsocketHeader(ctx)` (the `Authorization` header from `Client.AuthorizationHeader`, so token sources, token refresh, certificate tokens, `api.WithAuth` and the `api.WithTenant` override of Basic credentials apply; `User-Agent`; `api.WithExtraHeaders`; and what the request middleware changes) and `Client.DialWebsocket(ctx, url, header)`, which uses both and respects dry run. The upgrade request is passed through the request middleware (`ClientOptions.Middleware`, `Client.Use`) and only the headers it ends with are used. The response side of a middleware sees a synthetic `101` response, so middleware that needs the real response (retries, response logging) has no effect on a websocket. `wsdial.Connector` is the interface the websocket clients take; `*api.Client` implements it.
* **Fixed**: `Client.Realtime()` kept a snapshot of the credentials from the first call. It now reads the current authorization for every connect and reconnect, so a refreshed token is used, and it uses the client's TLS settings (it ignored `InsecureSkipVerify` and client certificates), proxy and middleware. The Bayeux handshake token comes from the same `Authorization` header as the upgrade request. `realtime.SetBearerToken` and `SetXSRFToken` still override it.
* **Fixed**: `notification2` `Service.CreateClient`/`SubscribeStream` and the remote access client used a default dialer (no client TLS settings, `websocket.DefaultDialer` for remote access) and, for remote access, `context.Background()`. They now use the client's dialer, headers and middleware. The notification2 connection does not send an `Authorization` header: the token in the URL authenticates it.
* **Fixed**: a websocket dialer copied from an HTTP transport which already made a request carried `h2` in its ALPN list, so the handshake with a server offering HTTP/2 failed. `WebsocketDialer` always negotiates HTTP/1.1.
* Added `realtime.Client.ConnectContext(ctx)` and `notification2.Notification2Client.ConnectContext(ctx)` (`Connect()` calls them with `context.Background()`). Cancelling `ctx` aborts connecting and, for realtime, the handshake wait (the client is closed and `ctx.Err()` is returned); automatic reconnects keep the context values but not its cancellation. Realtime also has `ClientOptions.Connector` and `SetConnector`; notification2 has `SetConnector`.
* Added remote access `RemoteAccessClient.DialContext`, `ListenServeContext`, `ServeContext` and `OpenMultiplexedSessionContext`, which take the caller's context (`ListenServe`, `Serve` and `OpenMultiplexedSession` keep their signature). `ServeContext` stops, and returns `ctx.Err()`, when `ctx` is cancelled.
* **Dry run for websockets**: with `api.WithDryRun(ctx, true)` nothing is connected. The upgrade request (`GET`, `Upgrade: websocket`, `Connection: Upgrade`, `Sec-WebSocket-Version: 13`, the client's headers) goes to the dry-run handler (`api.WithDryRunHandler`) or to a `DRY RUN` slog line with `Authorization` redacted, like an HTTP request. A `token` query parameter (notification2) is redacted. Behavior change: `SubscribeStream` and `Subscribe` of alarms, events, operations, measurements and managed objects, `realtime.Client.SubscribeStream` and `Notification2.SubscribeStream` used to connect for real even in dry run; they now return a result with `Status` `Skipped`, no error, `Request` set to the upgrade request, `Meta["channels"]` (`Meta["consumer"]` and `Meta["url"]` for notification2) and a stream that ends immediately without an error. `ConnectContext` returns an error for which `errors.Is(err, wsdial.ErrDryRun)` is true and `wsdial.DryRunRequest(err)` is the request. The remote access client's `DialContext` does the same, `ListenServeContext` and `ServeContext` return `nil` under dry run, and `RemoteAccessClient.DryRunRequest()` returns the request.
* Realtime reads the authorization again when the server asks for a new handshake on a connection which is kept, so a token which expired since is not sent again. This is what keeps a long-lived subscription of a client with a token source working.
* Added `Client.InvalidateToken()`, which discards the cached token of the token source so the next request gets a new one. The realtime client calls it before it reconnects and before it renews a handshake, so a token which the server revoked or expired is not sent again.
* **Fixed**: realtime closing the channel of a subscriber twice (a panic) when the client stopped after a reconnect, and a data race on the client id.
* **Fixed**: `notification2.Service.RenewToken` panicked on a token which is not a JWT; it now creates a new token.
* **Fixed**: `notification2.Service.SubscribeStream` leaked the client's goroutines when connecting failed.

### API coverage

* **Breaking**: `UIApplicationPlugins` (`ui/applicationplugins`) now manages the plugins in the application's `config.remotes` (`"<contextPath>@<version>": [modules]`), the model the Cumulocity UI uses. It wrote `applicationBuilder.plugins` before, which the UI does not read. Plugins are given as `"<name or contextPath>[@<version or tag>]"` and looked up among the tenant's own and the shared plugins:

  | Before | After |
  |---|---|
  | `List(ctx, app, ListOptions{})`, `app.Plugins() []map[string]any` | `List(ctx, app)`, `app.Plugins() []InstalledPlugin`; `ListPlugins(ctx, app)` lists them with the plugin details |
  | `Install(ctx, app, pluginID)` | `Install(ctx, app, plugins...)` |
  | `Update(ctx, app, []PluginReference)` / `Replace(...)` | `Update(ctx, app, UpdateOptions{Install, Remove, RemoveAll, UpdateAll, RemoveInvalid, Body})`, `Replace(ctx, app, plugins...)` |
  | `Delete(ctx, app, pluginID)` returning `op.Result[core.NoContent]` | `Delete(ctx, app, plugins...)` returning the updated application |

  Other `config` fields and `config.remotes` entries that are not plugins are kept. The application and plugin lookups are sent also in a dry run, so the dry run shows the complete update.
* Vendored the latest core OpenAPI spec. Added `Tenants.RestrictedRoles`, `SmartRestTemplates.BindExternalID`, `TrustedCertificates.CreateBulk`, `UnassignMany` on `ChildDevices`/`ChildAssets`/`ChildAdditions`, `TOTP.GetActivity`, and `Applications.DownloadBinary`.
* Added `Applications.DeployHosted`, which creates or updates a hosted web application from a folder (zipped with `pkg/zipUtilities`) or a `.zip` file, uploads the binary, and activates it. The name, key, and context path default to the `cumulocity.json` manifest, then to the file or folder name. `Owner` limits the lookup to applications owned by one tenant, `SkipUpload` and `SkipActivation` skip those steps, and `KeepBinaries` deletes the oldest binaries beyond that number (never the active or newly uploaded one). Also added `applications.ReadHostedManifest`. The fake server now stores application binaries (list, download, and delete) and filters applications by `owner`.
* Fixed `zipUtilities.ZipDirectoryContents`, `ZipDirectory`, and `ZipFile`: zip entry names now always use forward slashes (previously backslashes on Windows) and never start with "/", and errors from reading the source are returned instead of being ignored.
* Fixed `Applications.ListByOwner`, which called `/application/applicationsByTenant/{tenantId}` instead of `/application/applicationsByOwner/{tenantId}`.
* Fixed `zipUtilities.UnzipFile`, which extracted every entry except the given `names`; it now extracts only the named entries (case-insensitive), and everything when `names` is empty.
* Added `UserRoles.ResolveID`, which resolves a role id, self link or role document to the role id without a request, and a name with `*` or `?` wildcards (e.g. `ROLE_ALARM_*`) by matching the tenant's roles, also in a dry run. Added `Repository.Firmware.Patches.ResolveID`, like the `ResolveID` of the software and firmware versions.
* Added `RemoteAccess.Configurations.Detect`, which selects a device's passthrough configuration for a connection from an id, a name or a wildcard pattern, or automatically when none is given (the only one, else the one on port 22, else the first with "ssh" in its name).
* Fixed the remote access client's websocket handshake with basic auth, which sent the credentials without the tenant. It now sends the same `Authorization` header as a request of the client (`Client.AuthorizationHeader`), including a renewed token from the token source.
* The remote access client and `pkg/proxy` log to the Cumulocity client's logger (`ClientOptions.Logger`, now returned by `Client.Logger`) instead of `slog.Default()`, or to `RemoteAccessOptions.Logger`. Added `proxy.CopyWithLogger` and `proxy.CopyReadWriterWithLogger`.
* The fake server (`c8y-fakeserver`) serves the remote access websocket of a stored configuration, with an echo server as the device, so local proxies can be tested end to end.
* Added model accessors `Application.ContextPath`, `Application.OwnerTenantID`, `Tenant.Company`, `TrustedCertificate.Issuer` and `Notification2Subscription.SourceName`.
* The fake server matches `*` wildcards anywhere in an inventory query value (`name eq '*device*'`), not only at the end, and `GET /features` returns a plain JSON array like the platform (it returned a paged collection, which `Features.List` could not read).

### Errors and robustness

* **Fixed**: library code no longer exits or panics the host process on runtime failures. Each of these now returns an error:
  * `realtime`: an invalid host, a failed dial, a missing dialer, a rejected handshake (bad credentials) and a server advice of `none` used to call `os.Exit(1)` or `panic`. `Connect` now returns the error. A failure which happens later, in the background worker, stops the client, closes the channels of all subscribers and is returned from `Client.Err()` (new) and from pending acknowledgements. New errors: `realtime.ErrHandshakeFailed`, `ErrServerNoReconnect`, `ErrMissingDialer`.
  * `notification2`: an invalid host (`os.Exit`) and a missing dialer (`panic`). `Connect` returns the error. New `notification2.ErrMissingDialer`.
  * `authentication.DecodeBasicAuth` returns an error for a header without `user:password` instead of panicking.
  * `model.MeasurementSeriesGroup` and `MeasurementSeriesAggregateGroup` return an error from `UnmarshalJSON` for an invalid timestamp instead of panicking.
  * `microservice.WithServiceUserCredentials` ignores arguments after the first tenant instead of panicking.
* **Deprecated**, with a replacement which returns an error. The old functions keep their signature but no longer panic or exit:
  * `realtime.NewClient` -> `realtime.NewClientE`. With an invalid host the old function returns a client whose `Connect` returns the error.
  * `notification2.NewNotification2Client` -> `notification2.NewNotification2ClientE`. Same behavior for an invalid host.
  * `microservice.SetMicroserviceHost` -> `SetMicroserviceHostE`. The old method logs an invalid host and keeps the current one.
  * `encrypt.Encrypt`, `Decrypt`, `EncryptString`, `DecryptString` -> `EncryptE`, `DecryptE`, `EncryptStringE`, `DecryptStringE`. The old functions log the failure and return `nil` or `""`. `DecryptE` also returns `encrypt.ErrCiphertextTooShort` instead of panicking on truncated input.
* **Fixed**: an HTTP error response whose body is not JSON (an HTML page from a proxy, plain text, an empty body) keeps its HTTP status, so `errors.Is(err, api.ErrUnauthorized)` and the other status checks work. The message is the body text, or the status text when the body is empty or HTML.
* **Breaking (behavior, low impact)**: "not found" is now one concept. `op.ErrNotFound` is a 404 `*core.Error` and `api.ErrNotFound` is `api.Error{Code: 404}` (before: two unrelated `errors.New` values). `errors.Is(err, api.ErrNotFound)` is true for a real 404 response, for `Result.First()` on an empty collection and for `core.ErrNotFound(...)`. `errors.Is(err, op.ErrNotFound)` is now also true for a real 404, so `op.GetOrCreate` and `op.Upsert` treat a 404 from the lookup as "not found". Code which compared `err == op.ErrNotFound` by identity still works.
* Added `api.IsUnauthorized`, `api.IsForbidden`, `api.IsConflict` and `api.StatusCode(err) (int, bool)`. `api.IsNotFound` and `api.ErrHasStatus` now also work with the `api.Err*` sentinel values (they are `Error` values, not pointers).
* Docs: added an "Errors" section to the README and "Checking errors" to `docs/API_DESIGN.md`, and fixed examples there which used methods that do not exist (`Operations.Delete`, `ManagedObjects.GetOrCreate`).

### Tooling and documentation

* `c8ygen lint` now compares operations (method + path) and requires a reason for every drift waiver; waivers can name a method (`POST /path`).
* The README describes v2 (install, quick start, Go 1.27, stability promise), and the docs use `/v2` import paths. `docs/API_DESIGN.md` explains the tenant default for tenant-scoped endpoints and how `Applications.List` differs from the `ListBy*` methods.
* CI checks the docs (`task check-docs`: no v1 import paths, every `package main` example builds) and runs `gorelease` against the latest v2 tag (`task check-api-compat`, blocking once `v2.0.0` exists).
* The fake server (`c8y-fakeserver`) behaves more like a Cumulocity tenant, so go-c8y-cli tests can run against it: a json `POST` or `PUT` without an `Accept` header to a REST resource (inventory, alarms, events, measurements, operations, identity, users) returns no body; the tenant offers the `OAUTH2_INTERNAL` login option (initRequest `/tenant/oauth?tenant_id=t12345`) besides `BASIC`; and the `/tenant/oauth` token has `iat`, `nbf`, and `exp` claims.
* The fake server (`c8y-fakeserver`) covers more of the platform, so more go-c8y-cli manual test suites run against it:
  * Inventory queries support `bygroupid(<id>)` (the objects below a group, through child devices, assets and additions) and `not(<expression>)`; `withParents=true` also works on the managed object list, and each parent is the managed object itself (name, type, fragments), like the platform.
  * Binary downloads (`/inventory/binaries/{id}` and `/event/events/{id}/binaries`) send a `Content-Disposition` header with the file name, and `PUT /event/events/{id}/binaries` replaces the attachment.
  * Application versions: an uploaded zip is stored as a binary, the version defaults to the one in the zip's `cumulocity.json`, a tag (such as `latest`) belongs to a single version, `PUT .../versions/{version}` updates the tags, and `?version=` or `?tag=` only select a single version with the `application/vnd.com.nsn.cumulocity.applicationversion+json` Accept header (otherwise the whole collection is returned). Applications can be listed with `availability` and `hasVersions` (which embeds the versions). The shared `Cumulocity community plugins` plugin and the microservice `advanced-software-mgmt` with a running instance are seeded.
  * The standard web applications (cockpit, devicemanagement, administration) are owned by the management tenant, user groups have numeric ids, the global roles of a tenant are seeded, and the self link of a user escapes `@` as `%40`.
  * A request with the wrong password of the seeded `admin` or of a user created with a password is answered with `401`. Users do not return their password. Other users are still accepted with any password.
  * `c8y-fakeserver --tls` (`fakeserver.NewTLSServer`) serves https with a self-signed certificate, so the self links start with `https://` like on a tenant. Clients must accept the certificate.
* Fixed a flaky `Test_WatchSubscriptions_Lifecycle`: it counted the requests of the watcher around a lookup while the watcher polled every 20ms. The check is now its own test, `Test_WatchSubscriptions_NoRefreshOnMiss`, with a watcher that only polls once.

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
