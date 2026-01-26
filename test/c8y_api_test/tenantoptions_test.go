package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TenantOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// list
	collection, err := client.Tenants.Options.List(context.Background(), tenantoptions.ListOptions{})
	assert.NoError(t, err)
	assert.NotEmpty(t, collection.Self)

	// get
	option, err := client.Tenants.Options.Get(context.Background(), tenantoptions.GetOption{
		Category: collection.Options[0].Category,
		Key:      collection.Options[0].Key,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, collection.Self)
	assert.Equal(t, option.Category, collection.Options[0].Category)
	assert.Equal(t, option.Key, collection.Options[0].Key)

	// list by category
	category, err := client.Tenants.Options.ListByCategory(context.Background(), tenantoptions.ListByCategoryOptions{
		Category: collection.Options[0].Category,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(category), 1)
}
