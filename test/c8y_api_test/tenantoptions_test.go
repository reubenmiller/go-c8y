package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TenantOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// list
	collection := client.Tenants.Options.List(context.Background(), tenantoptions.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)

	// TODO: find a nicer way to access the first item in the results
	var firstOption model.TenantOption
	for item := range jsondoc.DecodeIter[model.TenantOption](collection.Data.Iter()) {
		firstOption = *item
		break
	}

	// get
	option := client.Tenants.Options.Get(context.Background(), tenantoptions.GetOption{
		Category: firstOption.Category,
		Key:      firstOption.Key,
	})
	assert.NoError(t, option.Err)
	assert.Equal(t, option.Data.Category(), firstOption.Category)
	assert.Equal(t, option.Data.Key(), firstOption.Key)

	// list by category
	category := client.Tenants.Options.ListByCategory(context.Background(), tenantoptions.ListByCategoryOptions{
		Category: firstOption.Category,
	})
	assert.NoError(t, category.Err)
	assert.GreaterOrEqual(t, len(category.Data), 1)
}
