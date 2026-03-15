package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TenantOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

	// list
	collection := client.Tenants.Options.List(context.Background(), tenantoptions.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)

	firstOption, err := collection.First()
	require.NoError(t, err)

	// get
	option := client.Tenants.Options.Get(context.Background(), tenantoptions.GetOption{
		Category: firstOption.Category(),
		Key:      firstOption.Key(),
	})
	assert.NoError(t, option.Err)
	assert.Equal(t, option.Data.Category(), firstOption.Category())
	assert.Equal(t, option.Data.Key(), firstOption.Key())

	// list by category
	category := client.Tenants.Options.ListByCategory(context.Background(), tenantoptions.ListByCategoryOptions{
		Category: firstOption.Category(),
	})
	assert.NoError(t, category.Err)
	assert.GreaterOrEqual(t, len(category.Data), 1)
}
