package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
)

// A missing resource is reported as an error which matches api.ErrNotFound,
// and the HTTP status is available from the result and from the error.
func ExampleIsNotFound() {
	client, stop := examplesetup.New()
	defer stop()

	result := client.Alarms.Get(context.Background(), "999999999")
	fmt.Println("status:", result.HTTPStatus)
	fmt.Println(errors.Is(result.Err, api.ErrNotFound), api.IsNotFound(result.Err))
	if code, ok := api.StatusCode(result.Err); ok {
		fmt.Println("code:", code)
	}

	// An empty collection is also "not found" when a single item is requested.
	empty := client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{Type: "no_such_type"})
	_, err := empty.First()
	fmt.Println(api.IsNotFound(err))
	// Output:
	// status: 404
	// true true
	// code: 404
	// true
}

// With api.WithDryRun the request is built and reported, but not sent. The
// handler receives the request that would have been sent.
func ExampleWithDryRun() {
	client, stop := examplesetup.New()
	defer stop()

	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithDryRunHandler(ctx, func(req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		fmt.Println(req.Method, req.URL.Path)
		fmt.Println(strings.TrimSpace(string(body)))
	})

	result := client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01"})
	// The request is kept in the result, and nothing was created.
	fmt.Println("error:", result.Err, "request:", result.Request.Method)
	// Output:
	// POST /inventory/managedObjects
	// {"name":"sensor-01"}
	// error: <nil> request: POST
}
