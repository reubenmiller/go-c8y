package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetFeatures(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// List features
	result := client.Features.List(ctx)

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	features, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(features), 0, "Feature array should not be empty")

	// Get single feature
	feature := features[0]
	getResult := client.Features.Get(ctx, feature.Key())

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, feature.Key(), getResult.Data.Key())
	assert.Equal(t, feature.Phase(), getResult.Data.Phase())
	assert.Equal(t, feature.Active(), getResult.Data.Active())
	assert.Empty(t, getResult.Data.TenantId(), "Feature tenant id should not be set")
}
