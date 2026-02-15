# go-c8y

Unofficial Go client for [Cumulocity IoT](https://cumulocity.com/api/core/).

[![tests](https://github.com/reubenmiller/go-c8y/actions/workflows/main.yml/badge.svg?branch=main)](https://github.com/reubenmiller/go-c8y/actions/workflows/main.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/reubenmiller/go-c8y)](https://goreportcard.com/report/github.com/reubenmiller/go-c8y)
[![Documentation c8y](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/c8y?status.svg)](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/c8y)
[![Documentation microservice ](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/microservice?status.svg)](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/microservice)

## v2 API (New!)

🎉 **The v2 API is under active development!** It features a modern, type-safe design with Go generics, rich result metadata, and flexible JSON handling.

**📖 [Read the complete v2 API Design Documentation](./docs/API_DESIGN.md)**

### Quick v2 Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/reubenmiller/go-c8y/pkg/c8y/api"
    "github.com/reubenmiller/go-c8y/pkg/c8y/api/operations"
)

func main() {
    // Create client
    client := api.NewClient(baseURL, username, password)
    ctx := context.Background()
    
    // List operations with smart device resolution
    result := client.Operations.List(ctx, operations.ListOptions{
        DeviceID: "name:my-device",  // Automatically resolved to ID
        Status:   "PENDING",
    })
    
    // Rich result handling
    if result.Err != nil {
        panic(result.Err)
    }
    
    // Iterate over operations
    for _, op := range result.Data.Items() {
        fmt.Printf("%s: %s\n", op.ID(), op.Status())
    }
}
```

### v2 Key Features

- **Rich Result Type** - Every operation returns metadata: status, HTTP details, duration, retry info
- **Flexible JSON Handling** - JSONDoc models with both structured and raw access
- **Context-Driven Behavior** - Dry run, deferred execution, and more via context
- **Smart Resolvers** - Reference devices by name, external ID, or query
- **Powerful Pagination** - Simple collections or iterators for large datasets

### v2 Documentation

- **[API_DESIGN.md](./docs/API_DESIGN.md)** - Complete API architecture and usage patterns
- **[V2.md](./docs/V2.md)** - Feature overview and implementation status
- **[IMPLEMENTATION_PATTERNS.md](./docs/IMPLEMENTATION_PATTERNS.md)** - Internal patterns (pipelines, upsert, retry)
- **[DEFERRED_EXECUTION.md](./docs/DEFERRED_EXECUTION.md)** - User confirmation prompts pattern
- **[DEVICE_RESOLVER_PATTERN.md](./docs/DEVICE_RESOLVER_PATTERN.md)** - Device resolution architecture
- **[REQUEST_INSPECTION.md](./docs/REQUEST_INSPECTION.md)** - Request debugging and security

---

## v1 API (Current Stable)

## v1 Caveats

We encourage you to try the package in your projects, just keep these caveats in mind, please:

* **This is a work in progress.** Not all of the Cumulocity IoT REST API is covered, and the HTTP client is very simple. In the future the HTTP Client will be improved to support retries on failures, client side rate limiting, prometheus api metrics, mqtt client.

* **There are no guarantees on API stability.** The general mechanics of the golang API are still being worked out. The balance between helpers and clarity is still being found. Given limited access to all available Cumulocity IoT versions, compatibility to all Cumulocity IoT versions is not guaranteed, however since Cumulocity IoT takes an additive approach to new features, it is more likely that the new features will be missing rather than existing API breaking (excluding deprecated features)

## v1 Usage

1. Add the package to your project using `go get`:

    ```sh
    go get -u github.com/reubenmiller/go-c8y
    ```

1. Create a `main.go` file with the following

    ```golang
    package main

    import (
        "context"
        "log/slog"

        "github.com/reubenmiller/go-c8y/pkg/c8y"
    )

    func main() {
        // Create the client from the following environment variables
        // C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
        client := c8y.NewClientFromEnvironment(nil, false)

        // Get list of alarms with MAJOR Severity
        alarmCollection, _, err := client.Alarm.GetAlarms(
            context.Background(),
            &c8y.AlarmCollectionOptions{
                Severity: "MAJOR",
            },
        )

        // Always check for errors
        if err != nil {
            slog.Error("Could not retrieve alarms", "err", err)
            panic(err)
        }

        slog.Info("Found alarms", "total", len(alarmCollection.Alarms))
    }
    ```

2. Set the credentials via environment variables

    **Windows (PowerShell)**

    ```sh
    $env:C8Y_HOST = "https://cumulocity.com"
    $env:C8Y_TENANT = "mycompany"
    $env:C8Y_USER = "username"
    $env:C8Y_PASSWORD = "password"
    ```

    **Linux/MacOS**

    ```sh
    export C8Y_HOST=https://cumulocity.com
    export C8Y_TENANT=mycompany
    export C8Y_USER=username
    export C8Y_PASSWORD=password
    ```

3. Run the application

    ```sh
    go run main.go
    ```

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
