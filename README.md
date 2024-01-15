# go-c8y

Unofficial Go client for [Cumulocity IoT](https://cumulocity.com/api/core/).

[![tests](https://github.com/reubenmiller/go-c8y/actions/workflows/main.yml/badge.svg)](https://github.com/reubenmiller/go-c8y/actions/workflows/main.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/reubenmiller/go-c8y)](https://goreportcard.com/report/github.com/reubenmiller/go-c8y)
[![Documentation c8y](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/c8y?status.svg)](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/c8y)
[![Documentation microservice ](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/microservice?status.svg)](https://godoc.org/github.com/reubenmiller/go-c8y/pkg/microservice)

## Caveats

We encourage you to try the package in your projects, just keep these caveats in mind, please:

* **This is a work in progress.** Not all of the Cumulocity IoT REST API is covered, and the HTTP client is very simple. In the future the HTTP Client will be improved to support retries on failures, client side rate limiting, prometheus api metrics, mqtt client.

* **There are no guarantees on API stability.** The general mechanics of the golang API are still being worked out. The balance between helpers and clarity is still being found. Given limited access to all available Cumulocity IoT versions, compatibility to all Cumulocity IoT versions is not guaranteed, however since Cumulocity IoT takes an additive approach to new features, it is more likely that the new features will be missing rather than existing API breaking (excluding deprecated features)

## Usage

1. Add the package to your project using `go get`:

    ```sh
    go get -u github.com/reubenmiller/go-c8y/c8y
    ```

1. Create a `main.go` file with the following

    ```golang
    package main

    import (
        "context"
        "log"

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
            log.Fatalf("Could not retrieve alarms. %s", err)
        }

        log.Printf("Total alarms: %d", len(alarmCollection.Alarms))
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
