package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
)

func Test_WithConflictAsError(t *testing.T) {
	client := newErrServer(t, 409, "application/json", `{"error":"identity/Duplicate","message":"exists"}`)
	ctx := context.Background()

	calls := map[string]func(ctx context.Context) (op.Status, int, error){
		"child devices assign": func(ctx context.Context) (op.Status, int, error) {
			r := client.ManagedObjects.ChildDevices.Assign(ctx, "1", "2")
			return r.Status, r.HTTPStatus, r.Err
		},
		"child assets assign": func(ctx context.Context) (op.Status, int, error) {
			r := client.ManagedObjects.ChildAssets.Assign(ctx, "1", "2")
			return r.Status, r.HTTPStatus, r.Err
		},
		"applications subscribe": func(ctx context.Context) (op.Status, int, error) {
			r := client.Applications.Subscribe(ctx, "t1", "2")
			return r.Status, r.HTTPStatus, r.Err
		},
	}
	for name, call := range calls {
		t.Run(name+" default is idempotent", func(t *testing.T) {
			status, code, err := call(ctx)
			assert.NoError(t, err)
			assert.Equal(t, op.StatusDuplicate, status)
			assert.Equal(t, 409, code)
		})
		t.Run(name+" with option returns the 409", func(t *testing.T) {
			status, code, err := call(api.WithConflictAsError(ctx))
			if assert.Error(t, err) {
				got, ok := api.StatusCode(err)
				assert.True(t, ok)
				assert.Equal(t, 409, got)
				assert.False(t, errors.Is(err, api.ErrNotFound))
			}
			assert.Equal(t, 409, code)
			assert.NotEqual(t, op.StatusDuplicate, status)
		})
	}
}
