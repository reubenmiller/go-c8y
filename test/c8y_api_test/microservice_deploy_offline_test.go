package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	c8yapi "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Microservices_Deploy_Offline(t *testing.T) {
	testcore.SkipLive(t)
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "svc" + testingutils.RandomString(6)
	manifest := filepath.Join(t.TempDir(), "cumulocity.json")
	require.NoError(t, os.WriteFile(manifest, []byte(`{"name":"`+name+`","requiredRoles":["ROLE_INVENTORY_READ"],"roles":[]}`), 0o600))

	// A dry run looks the microservice up for real but sends nothing else
	dry := client.Microservices.Deploy(c8yapi.WithDryRun(ctx, true), microservices.CreateOptions{File: manifest, Name: name})
	require.NoError(t, dry.Err)
	require.NotNil(t, dry.Request, "the dry run returns the prepared create request")
	assert.Equal(t, "POST", dry.Request.Method)
	_, err := client.Microservices.FindFirst(ctx, microservices.ListOptions{Name: name})
	assert.ErrorIs(t, err, op.ErrNotFound, "a dry run must not create the microservice")

	// A manifest (not a zip): the microservice is created and its roles set
	res := client.Microservices.Deploy(ctx, microservices.CreateOptions{File: manifest, Name: name, SkipSubscription: true})
	require.NoError(t, res.Err)
	t.Cleanup(func() {
		client.Microservices.Delete(ctx, res.Data.ID(), microservices.DeleteOptions{Force: true})
	})
	assert.Equal(t, op.StatusCreated, res.Status)
	assert.Equal(t, name, res.Data.Name())
	assert.Equal(t, `["ROLE_INVENTORY_READ"]`, res.Data.Get("requiredRoles").Raw)

	// Deploying again updates the existing microservice
	again := client.Microservices.Deploy(ctx, microservices.CreateOptions{File: manifest, Name: name, SkipSubscription: true})
	require.NoError(t, again.Err)
	assert.Equal(t, op.StatusUpdated, again.Status)
	assert.Equal(t, res.Data.ID(), again.Data.ID())
}
