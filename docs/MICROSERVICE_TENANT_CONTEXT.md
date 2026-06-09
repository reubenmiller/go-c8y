# Proposal: Multi-tenant request context and a framework-agnostic microservice package

Status: Implemented (branch `feat/microservice-tenant-context`)

## Motivation

The official Cumulocity **Java** microservice SDK has two capabilities that
were missing or only partially available in go-c8y:

1. **Per-request tenant context.** When the Cumulocity platform proxies a
   request to a microservice (`/service/{name}/...`), the request carries the
   caller's credentials. The Java SDK resolves these into a (thread-local)
   security context, and *all* downstream platform calls made while handling
   that request automatically execute on behalf of the caller's tenant. It
   also offers `MicroserviceSubscriptionsService.runForTenant(...)` /
   `runForEachTenant(...)` to execute arbitrary code within a tenant's
   context.

2. **Framework neutrality concerns (reversed).** The Java SDK forces Spring.
   go-c8y's `pkg/microservice` similarly forced consumers into specific
   frameworks: health endpoints were typed against `echo.Context`, so a user
   wanting chi, gorilla or plain `net/http` either pulled in echo anyway or
   reimplemented the handlers.

## Design

### 1. Context-scoped credentials in the core API layer

Go already has the right primitive for request-scoped state: `context.Context`
(the idiomatic equivalent of Java's thread-locals). The core `api` package now
supports carrying credentials in the context, honoured by every API call made
through a client:

```go
// Low-level: any credentials
ctx = api.WithAuth(ctx, authentication.AuthOptions{Tenant: "t123", Token: "..."})

// Convenience: a microservice service user
ctx = api.WithServiceUser(ctx, model.ServiceUser{Tenant: "t123", Username: "...", Password: "..."})

// Path-parameter only: override {tenantId} without changing credentials
ctx = api.WithTenant(ctx, "t123")

result := client.Devices.List(ctx, devices.ListOptions{})  // runs as t123
```

Implementation notes:

* `authentication.WithAuth` / `authentication.AuthFromContext` store and read
  the per-request `AuthOptions` (typed, unexported context key).
* A new client middleware (`MiddlewareContextAuthorization`, registered in
  `NewClient`) applies context credentials with priority over both the
  client-level credentials and the token source. Token selection follows the
  client's own precedence: `Token` (Bearer) first, then `Username`/`Password`
  (Basic, in `{tenant}/{user}` form). The `Tenant` field also overrides the
  `{tenantId}` path parameter used by tenant-scoped endpoints (e.g.
  `/user/{tenantId}/groups`).
* The token-source middleware skips requests that carry context credentials,
  so a shared client never "upgrades" a service-user request to the bootstrap
  token.
* The previous implementation of this idea lived as a private context key
  inside `pkg/microservice` and only worked for clients built via
  `NewBootstrapClient`. It is now a first-class capability of every client,
  which also benefits non-microservice use cases (e.g. management tools
  iterating subtenants).

#### Optional: per-tenant token exchange

By default, context credentials are sent as Basic auth on every request (the
same behaviour as the Java SDK). The client can optionally exchange them for
OAI-Secure bearer tokens, cached per tenant/user:

```go
client := api.NewClient(api.ClientOptions{..., ContextTokenExchange: true})
// or at runtime:
client.SetContextTokenExchange(true)
```

Behaviour:

* The first request for a tenant/user pays one login round-trip
  (`POST /tenant/oauth/token?tenant_id=...`); the token is then cached and
  refreshed automatically before expiry. Concurrent first-time requests for
  the same tenant are serialised so the login endpoint is not stampeded.
* If the exchange fails (e.g. the tenant does not support OAI-Secure), the
  request transparently falls back to Basic auth and the exchange is not
  re-attempted for that tenant/user for a cooldown period (5 minutes).
* If a cached token is rejected with a 401, it is invalidated and the request
  is retried once with the original Basic credentials.
* Credential rotation is detected (e.g. a service user recreated after an
  unsubscribe/subscribe cycle): a changed password resets the cache entry.

### 2. Incoming-request tenant context (Java SDK parity)

`pkg/microservice` gains a standard `net/http` middleware that mirrors the
Java SDK's per-request security context:

```go
ms := microservice.New(microservice.Options{})
_ = ms.Bootstrap(context.Background())

mux := http.NewServeMux()
requireTenant := ms.TenantContext() // func(http.Handler) http.Handler

mux.Handle("GET /devices", requireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Executes as the *caller's* tenant — no tenant plumbing in handler code
    result := ms.Client.Devices.List(r.Context(), devices.ListOptions{})
    ...
})))
```

* `AuthFromRequest(r)` extracts the caller's credentials: Basic auth with the
  `{tenant}/{username}` convention, or a Bearer token whose JWT `ten` claim
  identifies the tenant.
* `TenantContext(opts...)` validates the request and binds the credentials to
  `r.Context()`. Two scopes, matching the Java SDK's user/tenant scope split:
  * default — forward the **caller's own credentials** downstream (user
    scope; calls are limited by the caller's permissions).
  * `UseServiceUser: true` — only use the caller's credentials to identify
    the tenant, then run downstream calls as the tenant's **service user**
    (tenant scope, the roles requested in the manifest).
* Unauthenticated requests get a 401 (customisable via `OnError`).
* `TenantFromContext(ctx)` / `TenantFromRequest(r)` expose the caller's tenant
  to handler code.
* Because the middleware is a plain `func(http.Handler) http.Handler`, it
  works with `net/http`, chi, gorilla, and via adapters with echo
  (`echo.WrapMiddleware`) or gin.

For non-HTTP work (schedulers, queue consumers):

```go
// Java: subscriptionsService.runForEachTenant(...)
ms.ForEachTenant(ctx, func(ctx context.Context, user model.ServiceUser) error {
    result := ms.Client.Devices.List(ctx, devices.ListOptions{})
    return result.Err
})

// Java: contextService.runWithinContext(credentials, ...)
ctx := ms.WithServiceUser(ctx, "t123")
```

`WithServiceUser(parent, tenant...)` derives from a parent context (so
cancellation and deadlines propagate), unlike the older `ServiceUserContext`
which always started from `context.Background()`. `ServiceUserContext` is kept
and now delegates to it.

### 3. Simplifying pkg/microservice

* **echo removed from the module.** The health/env/logfile endpoints are now
  plain `net/http` handlers (`HealthHandler`, `EnvironmentVariablesHandler`,
  `GetLogFileHandler`, `PrometheusHandler`), with
  `RegisterHealthEndpoints(mux *http.ServeMux)` as a one-liner for the common
  case. Echo users can still mount them via `echo.WrapHandler`.
* **Construction / bootstrap split.** `New(opts)` builds the instance without
  contacting the platform; `Bootstrap(ctx)` loads application metadata and
  service users and returns an error instead of only logging.
  `NewDefaultMicroservice` remains as a deprecated wrapper with the previous
  log-and-continue behaviour.
* **cron removed from the module.** The `Scheduler` type is gone; users who
  need periodic work can use `time.Ticker` directly (see the multi-tenant
  example). The built-in operation polling
  (`StartOperationPolling`/`StopOperationPolling`) now uses a plain ticker and
  parses `agent.operations.pollRate` as a Go duration (`"30s"`); the legacy
  `"@every 30s"` form is still accepted, full cron expressions are not.
  Polling starts automatically during `RegisterMicroserviceAgent()` — the
  previous extra `Scheduler.Start()` call is no longer needed.
* Remaining dependencies (`viper` for configuration, `prometheus` for
  metrics) are kept: they are libraries rather than frameworks, do not leak
  into user-facing handler signatures, and the platform's conventions
  (`application.properties`, `/prometheus`) rely on them. They are candidates
  for future extraction into opt-in subpackages.

## Migration notes

| Before | After |
|---|---|
| `ms.AddHealthEndpointHandlers(e)` (echo) | `ms.RegisterHealthEndpoints(mux)`, or per-handler `e.GET("/health", echo.WrapHandler(http.HandlerFunc(ms.HealthHandler)))` |
| `NewDefaultMicroservice(opts)` | unchanged (deprecated) — prefer `New(opts)` + `Bootstrap(ctx)` |
| `ms.ServiceUserContext(tenant)` | unchanged — prefer `ms.WithServiceUser(ctx, tenant)` |
| manual loop over `ms.ServiceUsers` | `ms.ForEachTenant(ctx, fn)` |
| (not available) per-request tenant binding | `ms.TenantContext()` middleware |
