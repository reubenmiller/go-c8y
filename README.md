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

## Features

- Typed services for the Cumulocity core API, checked against the published OpenAPI spec in CI
- `op.Result[T]` results with metadata, pagination, and iterators
- JSON documents with both typed accessors and raw access
- Device resolvers: refer to a device by ID, name, external ID, or query
- Bulk managed object create and update, with partial-failure reporting and recovery
- Certificate login (RSA over HTTPS, or mTLS on port 8443) and enabling certificate authentication for users
- Multi-tenant microservices: subscription watching, per-tenant settings, Notification 2.0, and caller validation
- Context options for dry run, deferred execution, extra headers, and tenant selection
- HTTP middleware hooks and a file-backed response cache

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

More examples can be found under the `examples` folder.

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
