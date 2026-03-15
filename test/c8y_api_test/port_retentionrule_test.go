package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/retentionrules"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CRUD_RetentionRule(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create Retention rule
	createResult := client.RetentionRules.Create(ctx, model.RetentionRule{
		DataType:   "ALARM",
		MaximumAge: 10,
	})

	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.NotEmpty(t, createResult.Data.ID(), "Retention Rule should have a non-empty id")

	retentionRuleID := createResult.Data.ID()

	t.Cleanup(func() {
		client.RetentionRules.Delete(ctx, retentionRuleID)
	})

	// Get retention rule by id
	getResult := client.RetentionRules.Get(ctx, retentionRuleID)

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, retentionRuleID, getResult.Data.ID())

	// Get collection of data retention rules
	listResult := client.RetentionRules.List(ctx, retentionrules.ListOptions{})

	require.NoError(t, listResult.Err)
	assert.Equal(t, 200, listResult.HTTPStatus)

	rules, err := op.ToSliceR(listResult)
	require.NoError(t, err)
	assert.Greater(t, len(rules), 0, "Should have at least 1 data retention rule")

	// Delete retention rule
	deleteResult := client.RetentionRules.Delete(ctx, retentionRuleID)

	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}
