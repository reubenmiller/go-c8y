package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CRUD_TenantOption(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	category := "custom.app" + testingutils.RandomString(8)
	optionKey := "testKey"
	optionValue := "value1"

	// Create option
	createResult := client.Tenants.Options.Create(ctx, map[string]any{
		"category": category,
		"key":      optionKey,
		"value":    optionValue,
	})
	require.NoError(t, createResult.Err)
	assert.Equal(t, 200, createResult.HTTPStatus)
	assert.Equal(t, category, createResult.Data.Get("category").String())
	assert.Equal(t, optionKey, createResult.Data.Get("key").String())
	assert.Equal(t, optionValue, createResult.Data.Get("value").String())

	t.Cleanup(func() {
		client.Tenants.Options.Delete(ctx, tenantoptions.DeleteOptions{
			Category: category,
			Key:      optionKey,
		})
	})

	// Get option
	getResult := client.Tenants.Options.Get(ctx, tenantoptions.GetOption{
		Category: category,
		Key:      optionKey,
	})

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, category, getResult.Data.Get("category").String())
	assert.Equal(t, optionKey, getResult.Data.Get("key").String())
	assert.Equal(t, optionValue, getResult.Data.Get("value").String())

	// Update option
	optionValue2 := "value2"
	updateResult := client.Tenants.Options.Update(ctx, tenantoptions.UpdateOption{
		Category: category,
		Key:      optionKey,
		Body: map[string]any{
			"value": optionValue2,
		},
	})

	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, category, updateResult.Data.Get("category").String())
	assert.Equal(t, optionKey, updateResult.Data.Get("key").String())
	assert.Equal(t, optionValue2, updateResult.Data.Get("value").String())
}

func Test_CRUD_TenantOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	settings := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
		"prop3": "value3",
	}

	category := "custom.ci.multi" + testingutils.RandomString(8)

	// Update multiple options
	updateResult := client.Tenants.Options.UpdateByCategory(ctx, tenantoptions.UpdateByCategoryOption{
		Category: category,
		Body:     settings,
	})

	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, "value1", updateResult.Data["prop1"])
	assert.Equal(t, "value2", updateResult.Data["prop2"])
	assert.Equal(t, "value3", updateResult.Data["prop3"])

	t.Cleanup(func() {
		for key := range settings {
			client.Tenants.Options.Delete(ctx, tenantoptions.DeleteOptions{
				Category: category,
				Key:      key,
			})
		}
	})

	// Get Options for a category
	getResult := client.Tenants.Options.ListByCategory(ctx, tenantoptions.ListByCategoryOptions{
		Category: category,
	})

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, "value1", getResult.Data["prop1"])
	assert.Equal(t, "value2", getResult.Data["prop2"])
	assert.Equal(t, "value3", getResult.Data["prop3"])
}

func Test_GetTenantOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	category := "ciMulti" + testingutils.RandomString(8)

	// Update multiple options
	settings := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}
	updateResult := client.Tenants.Options.UpdateByCategory(ctx, tenantoptions.UpdateByCategoryOption{
		Category: category,
		Body:     settings,
	})
	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)

	t.Cleanup(func() {
		for key := range settings {
			client.Tenants.Options.Delete(ctx, tenantoptions.DeleteOptions{
				Category: category,
				Key:      key,
			})
		}
	})

	// Get options by category
	getByCategoryResult := client.Tenants.Options.ListByCategory(ctx, tenantoptions.ListByCategoryOptions{
		Category: category,
	})
	require.NoError(t, getByCategoryResult.Err)
	assert.Equal(t, 200, getByCategoryResult.HTTPStatus)
	assert.Equal(t, "value1", getByCategoryResult.Data["prop1"])
	assert.Equal(t, "value2", getByCategoryResult.Data["prop2"])

	// Get all options and filter by category on the client side
	allOptions := client.Tenants.Options.List(
		context.Background(),
		tenantoptions.ListOptions{
			PaginationOptions: pagination.NewPaginationOptions(100),
		},
	)
	assert.NoError(t, allOptions.Err)
	assert.Equal(t, http.StatusOK, allOptions.HTTPStatus)
	assert.GreaterOrEqual(t, allOptions.Data.Length(), 2, "Should be at least 2 options")

	filteredOptions := map[string]string{}
	for opt := range op.Iter(allOptions) {
		if opt.Category() == category {
			filteredOptions[opt.Key()] = opt.Value()
		}
	}

	testingutils.Equals(t, "value1", filteredOptions["prop1"])
	testingutils.Equals(t, "value2", filteredOptions["prop2"])
}
