# go-c8y

Unofficial Go client for [Cumulocity](http://cumulocity.com/guides/reference/rest-implementation/).

## Caveats

We encourage you to try the package in your projects, just keep these caveats in mind, please:

* **This is a work in progress.** Not all of the Cumulocity REST API is covered, and the HTTP client is very simple. In the future the HTTP Client will be improved to support retries on failures, client side rate limiting, prometheus api metrics, mqtt client.

* **There are no guarantees on API stability.** The general mechanics of the golang API are still being worked out. The balance between helpers and clarity is still being found. Given limited access to all available Cumulocity Servers versions, compatibility to all Cumulocity versions is not guarenteed, however since Cumulocity takes an additive approach to new features, it is more likely that the new features will be missing rather than existing API breaking (excluding deprecated features)



## Installation

Install the package with `go get`:

    go get -u github.com/reubenmiller/go-c8y/c8y_test

Or, add the package to your `go.mod` file:

    require github.com/reubenmiller/go-c8y/c8y_test master

Or, clone the repository:

    git clone https://github.com/reubenmiller/go-c8y/c8y_test.git && cd go-c8y


## Running the tests

1. Create an `application.properties` file in the `./c8y_test` folder and set the Cumulocity host, tenant and user credentials. The user must have admin rights in the specified tenant.

    ```sh
    c8y.host=
    c8y.tenant=
    c8y.username=
    c8y.password=
    testing.cleanup.removeDevice=false
    ```

2. Execute the integration tests

    ```sh
    go test -timeout 30s github.com/reubenmiller/go-c8y/c8y_test
    ```

## Usage

1. Create a `main.go` file with the following

    ```golang
    package main

    import (
        "context"
        "log"

        c8y "github.com/reubenmiller/go-c8y"
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

2. Execute the application

    On Windows
    ```sh
    set C8Y_HOST=https://cumulocity.com
    set C8Y_TENANT=mycompany
    set C8Y_USER=username
    set C8Y_PASSWORD=password
    go run main.go
    ```

    On Linux/MacOS
    ```sh
    export C8Y_HOST=https://cumulocity.com
    export C8Y_TENANT=mycompany
    export C8Y_USER=username
    export C8Y_PASSWORD=password
    go run main.go
    ```

## Examples

More examples can be found under the `examples` folder.
