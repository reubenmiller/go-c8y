# CHANGELOG

## Unreleased

* Added context-scoped credentials to the core API client: `api.WithAuth`, `api.WithServiceUser` and `api.WithTenant` allow any request made with a shared client to execute on behalf of a specific tenant (see `docs/MICROSERVICE_TENANT_CONTEXT.md`)
* Added `Microservice.TenantContext()`, a framework-agnostic `net/http` middleware that binds the caller's tenant credentials to the request context (parity with the Java SDK's per-request security context), with optional service-user (tenant) scope via `TenantContextOptions{UseServiceUser: true}`
* Added `Microservice.ForEachTenant()` and `Microservice.WithServiceUser()` helpers for running code within a tenant's context (parity with the Java SDK's `runForEachTenant`/`runForTenant`)
* Added `microservice.New()` + `Microservice.Bootstrap(ctx)` for explicit, error-returning startup; `NewDefaultMicroservice` is deprecated but unchanged
* **Breaking**: `pkg/microservice` no longer depends on the echo web framework. The health endpoints are now plain `net/http` handlers; use `RegisterHealthEndpoints(mux)` or mount the handlers individually (echo users can wrap them with `echo.WrapHandler`)
* Added optional per-tenant token exchange: `api.ClientOptions{ContextTokenExchange: true}` (or `client.SetContextTokenExchange(true)`) exchanges per-request basic credentials for cached OAI-Secure bearer tokens, with automatic refresh, 401 invalidation and a basic-auth fallback when the tenant does not support OAI-Secure
* **Breaking**: `pkg/microservice` no longer depends on the cron library and the `Scheduler` type has been removed — use a `time.Ticker` for periodic work. Operation polling (`StartOperationPolling`, stoppable via the new `StopOperationPolling`) uses a plain ticker and starts automatically during `RegisterMicroserviceAgent()`; `agent.operations.pollRate` accepts Go durations (`"30s"`) and the legacy `"@every 30s"` form, but no longer full cron expressions

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
