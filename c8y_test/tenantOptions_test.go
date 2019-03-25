package c8y_test

import (
	"context"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestTenantOptionsService_CRUD_Option(t *testing.T) {
	client := createTestClient()

	category := "custom.app"
	optionKey := "testKey"
	optionValue := "value1"

	//
	// Create option
	option, resp, err := client.TenantOptions.Create(
		context.Background(),
		&c8y.TenantOption{
			Category: category,
			Key:      optionKey,
			Value:    optionValue,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, category, option.Category)
	testingutils.Equals(t, optionKey, option.Key)
	testingutils.Equals(t, optionValue, option.Value)

	//
	// Get option
	option2, resp, err := client.TenantOptions.GetOption(
		context.Background(),
		category,
		optionKey,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, category, option2.Category)
	testingutils.Equals(t, optionKey, option2.Key)
	testingutils.Equals(t, optionValue, option2.Value)

	//
	// Update option
	optionValue2 := "value2"
	option3, resp, err := client.TenantOptions.Update(
		context.Background(),
		category,
		optionKey,
		optionValue2,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, category, option3.Category)
	testingutils.Equals(t, optionKey, option3.Key)
	testingutils.Equals(t, optionValue2, option3.Value)
}

func TestTenantOptionsService_CRUD_Options(t *testing.T) {
	client := createTestClient()

	settings := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
		"prop3": "value3",
	}

	category := "custom.ci.multi"

	//
	// Update multiple options
	options, resp, err := client.TenantOptions.UpdateOptions(
		context.Background(),
		category,
		settings,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "value1", options["prop1"])
	testingutils.Equals(t, "value2", options["prop2"])
	testingutils.Equals(t, "value3", options["prop3"])

	//
	// Get Options for a category
	options2, resp, err := client.TenantOptions.GetOptionsForCategory(
		context.Background(),
		category,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "value1", options2["prop1"])
	testingutils.Equals(t, "value2", options2["prop2"])
	testingutils.Equals(t, "value3", options2["prop3"])
}

func TestTenantOptionsService_GetOptions(t *testing.T) {
	client := createTestClient()

	category := "ciMulti2"

	//
	// Update multiple options
	settings := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}
	_, resp, err := client.TenantOptions.UpdateOptions(
		context.Background(),
		category,
		settings,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)

	//
	// Update multiple options
	allOptions, resp, err := client.TenantOptions.GetOptions(
		context.Background(),
		&c8y.PaginationOptions{
			PageSize: 100,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(allOptions.Options) >= 2, "Should be at least 2 options")

	filteredOptions, resp, err := client.TenantOptions.GetOptionsForCategory(
		context.Background(),
		category,
	)

	/*
		// TODO: Switch to using .GetOptions, but it does not work at the moment (c8y bug)
		filteredOptions := map[string]string{}
		for _, opt := range allOptions.Options {
			if opt.Category == category {
				filteredOptions[opt.Key] = opt.Value
			}
		}
	*/
	testingutils.Equals(t, "value1", filteredOptions["prop1"])
	testingutils.Equals(t, "value2", filteredOptions["prop2"])
}
