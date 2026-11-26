package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
)

func Test_WithNotFoundAsError_Delete(t *testing.T) {
	client := newErrServer(t, 404, "application/json", `{"error":"inventory/notFound","message":"nope"}`)
	ctx := context.Background()

	calls := map[string]func(ctx context.Context) op.Result[any]{
		"events delete": func(ctx context.Context) op.Result[any] {
			r := client.Events.Delete(ctx, "1")
			return op.Result[any]{Err: r.Err, Status: r.Status, HTTPStatus: r.HTTPStatus}
		},
		"binaries delete": func(ctx context.Context) op.Result[any] {
			r := client.Binaries.Delete(ctx, "1")
			return op.Result[any]{Err: r.Err, Status: r.Status, HTTPStatus: r.HTTPStatus}
		},
		"managedobjects delete": func(ctx context.Context) op.Result[any] {
			r := client.ManagedObjects.Delete(ctx, "1", managedobjects.DeleteOptions{})
			return op.Result[any]{Err: r.Err, Status: r.Status, HTTPStatus: r.HTTPStatus}
		},
		"child devices unassign": func(ctx context.Context) op.Result[any] {
			r := client.ManagedObjects.ChildDevices.Unassign(ctx, "1", "2")
			return op.Result[any]{Err: r.Err, Status: r.Status, HTTPStatus: r.HTTPStatus}
		},
	}
	for name, call := range calls {
		t.Run(name+" default is idempotent", func(t *testing.T) {
			r := call(ctx)
			assert.NoError(t, r.Err)
			assert.Equal(t, op.StatusSkipped, r.Status)
			assert.Equal(t, 404, r.HTTPStatus)
		})
		t.Run(name+" with option returns the 404", func(t *testing.T) {
			r := call(api.WithNotFoundAsError(ctx))
			if assert.Error(t, r.Err) {
				assert.True(t, errors.Is(r.Err, api.ErrNotFound))
				assert.True(t, api.IsNotFound(r.Err))
				assert.Contains(t, r.Err.Error(), "nope")
			}
			assert.Equal(t, 404, r.HTTPStatus)
			assert.NotEqual(t, op.StatusSkipped, r.Status)
		})
	}
}

func Test_WithNotFoundAsError_OtherErrorsUnchanged(t *testing.T) {
	client := newErrServer(t, http.StatusForbidden, "application/json", `{"error":"x","message":"denied"}`)
	r := client.Events.Delete(api.WithNotFoundAsError(context.Background()), "1")
	assert.Error(t, r.Err)
	assert.Equal(t, 403, r.HTTPStatus)
}

// Deletes ignore a 404 by default. api.WithNotFoundAsError keeps it as an error.
func ExampleWithNotFoundAsError() {
	client, stop := examplesetup.New()
	defer stop()

	r := client.Events.Delete(context.Background(), "999999999")
	fmt.Println("default:", r.Err, r.Status)

	ctx := api.WithNotFoundAsError(context.Background())
	r = client.Events.Delete(ctx, "999999999")
	fmt.Println("option:", api.IsNotFound(r.Err), r.HTTPStatus)
	// Output:
	// default: <nil> Skipped
	// option: true 404
}
