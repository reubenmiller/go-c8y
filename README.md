# go-c8y

Unofficial Go client for the [Cumulocity](https://cumulocity.com/api/core/) REST API.

[![tests](https://github.com/reubenmiller/go-c8y/actions/workflows/v2.yml/badge.svg?branch=v2)](https://github.com/reubenmiller/go-c8y/actions/workflows/v2.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/reubenmiller/go-c8y/v2.svg)](https://pkg.go.dev/github.com/reubenmiller/go-c8y/v2/pkg/c8y/api)

## Requirements

- Go 1.27 or later. The SDK uses generic methods, which Go 1.27 introduced.

## Install

```sh
go get github.com/reubenmiller/go-c8y/v2
```

## Quick start

Set the connection details as environment variables:

```sh
export C8Y_HOST=https://example.cumulocity.com
export C8Y_TENANT=t12345
export C8Y_USER=username
export C8Y_PASSWORD=password
```

Then list the major alarms:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
)

func main() {
	// Reads C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD (or C8Y_TOKEN)
	client := api.NewClientFromEnvironment(api.ClientOptions{})

	result := client.Alarms.List(context.Background(), alarms.ListOptions{
		Severity: []model.AlarmSeverity{model.AlarmSeverityMajor},
	})
	if result.Err != nil {
		fmt.Fprintln(os.Stderr, "could not list alarms:", result.Err)
		os.Exit(1)
	}

	for alarm, err := range result.Items() {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(alarm.Get("id").String(), alarm.Get("text").String())
	}
}
```

Every call returns an `op.Result[T]` with the data, the error, the HTTP status, and request metadata such as duration and retry attempts.

## Errors

Check errors with `errors.Is` against the sentinels in the `api` package, or with the helpers. The HTTP status is available for every error response, also when the body is not JSON.

```go
result := client.Alarms.Get(ctx, "12345")
if errors.Is(result.Err, api.ErrNotFound) { // or api.IsNotFound(result.Err)
	// no such alarm; also true for Result.First() on an empty collection
}
if api.IsUnauthorized(result.Err) || api.IsForbidden(result.Err) || api.IsConflict(result.Err) {
	// ...
}
if code, ok := api.StatusCode(result.Err); ok {
	fmt.Println("HTTP status:", code)
}
```

See "Error Handling" in [API_DESIGN.md](./docs/API_DESIGN.md) for details.

## Pagination

`List` returns one page. `ListAll` returns an iterator which fetches the next page when needed. The page size, the maximum number of items, the paging strategy and the number of parallel page requests are set with the embedded `PaginationOptions`:

```go
opts := managedobjects.ListOptions{
	Type:              "c8y_Device",
	PaginationOptions: pagination.PaginationOptions{PageSize: 100, MaxItems: 1000},
}
for mo, err := range client.ManagedObjects.ListAll(ctx, opts).Items() {
	if err != nil {
		return err // a failed page request ends the iteration
	}
	fmt.Println(mo.Get("id").String(), mo.Get("name").String())
}
```

`pipeline` has helpers to filter, batch and process the items with bounded concurrency.

## Dry run

`api.WithDryRun` builds and reports a request without sending it. The request is kept in `result.Request`, or passed to a handler:

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
)

func main() {
	client := api.NewClientFromEnvironment(api.ClientOptions{})

	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithDryRunHandler(ctx, func(req *http.Request) {
		fmt.Println(req.Method, req.URL.Path)
	})

	result := client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01"})
	fmt.Println("error:", result.Err)
}
```

Dry run also applies to realtime, Notification 2.0 and remote access: nothing connects, and the websocket upgrade request is reported the same way.

## Request hooks, progress and raw requests

`api.WithRequestHook` runs a function on the final `*http.Request` of every call made with the context. `api.WithUploadProgress` and `api.WithDownloadProgress` report the bytes transferred, and `UploadFileOptions.Progress` reports the file bytes of a typed upload. `Client.Request` calls an endpoint which has no typed method and returns an `op.Result[jsondoc.JSONDoc]`:

```go
ctx = api.WithRequestHook(ctx, func(req *http.Request) error {
	req.Header.Set("X-Trace", "abc")
	return nil
})
result := client.Request(ctx, api.RequestOptions{Method: http.MethodGet, Path: "/inventory/managedObjects/12345"})
```

## Realtime, Notification 2.0 and remote access

`SubscribeStream` on alarms, events, measurements, operations and managed objects connects to the realtime API and returns a typed stream. Notification 2.0 has its own service (`client.Notification2`) for subscriptions and tokens, and a stream for consuming them. Remote access is the REST API for the configurations (`client.RemoteAccess.Configurations`) plus the websocket proxy in `pkg/remoteaccess`:

```go
result := client.Alarms.SubscribeStream(ctx, "12345") // a device id
if result.Err != nil {
	return result.Err
}
stream := result.Data
defer stream.Close()
for message, err := range stream.Items() {
	if err != nil {
		return err
	}
	fmt.Println(message.Action, message.Data.Text())
}
```

All three use the TLS, proxy and authentication settings of the client.

## Features

- Typed services for the Cumulocity core API, checked against the published OpenAPI spec in CI: inventory (managed objects, devices, groups, child references), alarms, events, measurements, operations and bulk operations, binaries, identity, audit records, retention rules
- Device and software management: device registration, certificate enrollment, the repository (software, firmware and configuration, with versions and patches), device profiles, Device Data Hub, data broker, digital twin (DTM) and data preparation
- Platform administration: tenants and tenant options, applications and application versions, microservices, UI plugins, users, groups, roles and inventory roles, login options, trusted certificates and the certificate authority, feature toggles, usage statistics
- `op.Result[T]` results with metadata, an iterator for every `ListAll`, and `errors.Is` support with `api.IsNotFound`, `api.IsConflict` and other helpers
- JSON documents with both typed accessors and raw access
- Device resolvers: refer to a device by ID, name, external ID, or query. `ResolveID` for managed objects, users, tenants, binaries and the repository
- Bulk managed object create and update, with partial-failure reporting and recovery, and `pipeline` helpers for streams of items
- Realtime, Notification 2.0 and Cloud Remote Access over websockets, with a shared dialer, authentication and dry run
- Authentication: basic, token, OAuth2 and SSO (browser and device flow), TFA callbacks, certificate login (RSA over HTTPS, or mTLS on port 8443) and enabling certificate authentication for users
- Multi-tenant microservices: subscription watching, per-tenant settings, Notification 2.0, and caller validation
- Context options for dry run, deferred execution, request approval, extra headers, extra query parameters, tenant selection and processing mode
- Request hooks, upload and download progress, HTTP middleware, `Client.Request` for endpoints without a typed method, and a file-backed response cache
- An in-process fake server (`internal/pkg/fakeserver`) for offline tests: see [OFFLINE_TESTING.md](./docs/OFFLINE_TESTING.md)

## API stability

v2 follows [semantic versioning](https://semver.org). From `v2.0.0` on, exported identifiers in this module are not removed or changed incompatibly within the v2 major version. Packages under `internal/` are not part of the API. CI runs an API compatibility check against the latest v2 tag on every pull request.

Releases before `v2.0.0` (pseudo-versions and pre-releases) may still change.

The v1 module (`github.com/reubenmiller/go-c8y`, `v0.x` tags) is a separate module. See the [CHANGELOG](./CHANGELOG.md) for migration notes.

## Documentation

- [API_DESIGN.md](./docs/API_DESIGN.md): API architecture and usage patterns
- [V2.md](./docs/V2.md): feature overview
- [MICROSERVICE_TENANT_CONTEXT.md](./docs/MICROSERVICE_TENANT_CONTEXT.md): multi-tenant microservices
- [DEFERRED_EXECUTION.md](./docs/DEFERRED_EXECUTION.md): confirmation prompts before a request runs
- [DEVICE_RESOLVER_PATTERN.md](./docs/DEVICE_RESOLVER_PATTERN.md): device resolution
- [REQUEST_INSPECTION.md](./docs/REQUEST_INSPECTION.md): request debugging and redaction
- [IMPLEMENTATION_PATTERNS.md](./docs/IMPLEMENTATION_PATTERNS.md): pipelines, upsert, and retry
- [OFFLINE_TESTING.md](./docs/OFFLINE_TESTING.md): testing against the fake server

## Examples

Every service package has a package comment with a usage snippet (see [pkg.go.dev](https://pkg.go.dev/github.com/reubenmiller/go-c8y/v2/pkg/c8y/api)). These `Example` functions run offline against the fake server as part of `go test`:

- [managed objects](./pkg/c8y/api/inventory/managedobjects/example_test.go): create, get, `ListAll`, `ResolveID`
- [alarms](./pkg/c8y/api/alarms/example_test.go), [events](./pkg/c8y/api/events/example_test.go), [measurements](./pkg/c8y/api/measurements/example_test.go), [operations](./pkg/c8y/api/operations/example_test.go)
- [software item and version](./pkg/c8y/api/repository/software/example_test.go)
- [applications](./pkg/c8y/api/applications/example_test.go)
- [users and `ResolveID` with a wildcard](./pkg/c8y/api/users/example_test.go)
- [pagination](./pkg/c8y/api/pagination/example_test.go)
- [errors and dry run](./pkg/c8y/api/example_errors_dryrun_test.go)
- [request hook and upload progress](./pkg/c8y/api/example_request_hook_test.go)
- [realtime under dry run](./pkg/c8y/api/example_websocket_test.go)
- [authentication](./pkg/c8y/api/example_oauth2_test.go)

Complete programs can be found under the [examples](./examples) folder.

## Development

### Running the tests

To run the tests you will need to install [go-task](https://taskfile.dev/installation/), or use the dev container which includes go-task.

1. Create a dotenv file `.env` to the root folder and add your Cumulocity credentials to use for the tests

    ```sh
    C8Y_HOST=https://cumulocity.com
    C8Y_TENANT=mycompany
    C8Y_USER=username
    C8Y_PASSWORD=password
    ```

2. Execute the tests

    ```sh
    task test
    ```
