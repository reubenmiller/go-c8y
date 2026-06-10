# Multi-tenant microservice example

A `MULTI_TENANT` Cumulocity microservice built with only the Go standard
library's `net/http` server — no web framework required.

It demonstrates the tenant-context features of the SDK:

* **Incoming requests** — `ms.TenantContext()` is a standard
  `func(http.Handler) http.Handler` middleware. It reads the credentials the
  Cumulocity platform forwards with each request (Basic auth in
  `{tenant}/{user}` format, or a Bearer token) and binds them to the request
  context. Every API call made with `r.Context()` then automatically executes
  on behalf of the caller's tenant — equivalent to the per-request security
  context in the Cumulocity Java SDK.

* **Tenant scope vs user scope** — pass
  `microservice.TenantContextOptions{UseServiceUser: true}` to run downstream
  calls as the tenant's *service user* (with the roles from the manifest)
  instead of forwarding the caller's own credentials.

* **Background jobs** — `ms.ForEachTenant(ctx, fn)` runs `fn` once per
  subscribed tenant with a context carrying that tenant's service user,
  equivalent to the Java SDK's
  `MicroserviceSubscriptionsService.runForEachTenant()`.

## Endpoints

| Endpoint      | Description                                            |
|---------------|--------------------------------------------------------|
| `GET /devices` | Lists devices of the *caller's* tenant                |
| `GET /health`  | Health endpoint used by the platform probes           |
| `GET /env`     | Configuration and environment variables (masked)      |
| `GET /prometheus` | Prometheus metrics                                 |

## Build and deploy

```sh
./build.sh
# Then upload multi-tenant-demo.zip via the Cumulocity Administration application
```

## Using another router

The middleware works with any router that accepts `net/http` middleware:

```go
// chi
r := chi.NewRouter()
r.Use(ms.TenantContext())

// echo
e := echo.New()
e.Use(echo.WrapMiddleware(ms.TenantContext()))
```
