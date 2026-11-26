package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RawBody of a real collection response: the envelope as received (with
// statistics), or a top-level JSON array, and nothing for a dry run.
func Test_RawBody_Collections(t *testing.T) {
	testcore.SkipLive(t)
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	typ := "rawbody_" + testingutils.RandomString(8)
	created := client.ManagedObjects.Create(ctx, map[string]any{"name": "rawbody", "type": typ})
	require.NoError(t, created.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(ctx, created.Data.ID(), managedobjects.DeleteOptions{})
	})

	t.Run("envelope", func(t *testing.T) {
		res := client.ManagedObjects.List(ctx, managedobjects.ListOptions{Type: typ})
		require.NoError(t, res.Err)
		body := res.RawBody()
		assert.Equal(t, op.BodyEnvelope, body.Kind)
		assert.False(t, body.Empty)
		require.True(t, body.Has())
		assert.Equal(t, string(res.Response), string(body.Raw))
		assert.Len(t, body.Doc().Get("managedObjects").Array(), 1)
		assert.True(t, body.Doc().Get("statistics").Exists())
	})

	t.Run("empty envelope", func(t *testing.T) {
		res := client.ManagedObjects.List(ctx, managedobjects.ListOptions{Type: typ + "_none"})
		require.NoError(t, res.Err)
		body := res.RawBody()
		assert.Equal(t, op.BodyEnvelope, body.Kind)
		assert.True(t, body.Empty)
		assert.True(t, body.Doc().Get("managedObjects").IsArray())
	})

	t.Run("dry run", func(t *testing.T) {
		res := client.ManagedObjects.List(api.WithDryRun(ctx, true), managedobjects.ListOptions{Type: typ})
		require.NoError(t, res.Err)
		assert.Equal(t, op.Body{}, res.RawBody())
	})

	t.Run("single object", func(t *testing.T) {
		res := client.ManagedObjects.Get(ctx, created.Data.ID(), managedobjects.GetOptions{})
		require.NoError(t, res.Err)
		assert.Equal(t, op.Body{}, res.RawBody())
	})

	t.Run("top-level array", func(t *testing.T) {
		res := client.Tenants.UsageStatistics.ListSummaryAllTenants(ctx, usagestatistics.ListSummaryAllTenantsOptions{
			DateFrom: time.Now().AddDate(-1, 0, 0),
		})
		require.NoError(t, res.Err)
		body := res.RawBody()
		assert.Equal(t, op.BodyArray, body.Kind)
		assert.True(t, body.Doc().Get().IsArray())
	})
}
