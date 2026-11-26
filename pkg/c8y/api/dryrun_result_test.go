package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Under dry run nothing is sent: the result is Skipped (not Created/NoContent),
// has no error, and carries the request that would have been sent.
func Test_DryRun_ResultIsSkippedWithRequest(t *testing.T) {
	client, stop := examplesetup.New()
	defer stop()

	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithMockResponses(ctx, false)

	created := client.Alarms.Create(ctx, alarms.CreateOptions{})
	require.NoError(t, created.Err)
	assert.Equal(t, op.StatusSkipped, created.Status)
	assert.True(t, created.IsSuccess())
	require.NotNil(t, created.Request)
	assert.Equal(t, "POST", created.Request.Method)
	assert.Contains(t, created.Meta["reason"], "dry run")

	deleted := client.ManagedObjects.Delete(ctx, "123", managedobjects.DeleteOptions{})
	require.NoError(t, deleted.Err)
	assert.Equal(t, op.StatusSkipped, deleted.Status)
	require.NotNil(t, deleted.Request)
	assert.Equal(t, "DELETE", deleted.Request.Method)
}
