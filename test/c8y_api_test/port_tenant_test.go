package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetTenantStatisticsSummary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	dateTo := time.Now()
	dateFrom := dateTo.Add(-1 * 10 * 24 * time.Hour)

	result := client.Tenants.UsageStatistics.ListSummary(ctx, usagestatistics.ListSummaryOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.GreaterOrEqual(t, result.Data.Get("storageSize").Int(), int64(0), "Storage size should be greater than or equal to 0")
	assert.Greater(t, result.Data.Get("requestCount").Int(), int64(0), "Request count should be greater than 0")
}

func Test_GetTenantStatistics(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	dateTo := time.Now()
	dateFrom := dateTo.Add(-1 * 10 * 24 * time.Hour)

	result := client.Tenants.UsageStatistics.List(ctx, usagestatistics.ListOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	statistics, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(statistics), 10, "At most 10 days should be returned")
	assert.Greater(t, len(statistics), 0, "At least 1 day should be returned")
}

func Test_GetAllTenantsStatisticsSummary(t *testing.T) {
	// TODO: Test requires a tenant with subtenant capabilities
	t.SkipNow()
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	dateTo := time.Now()
	dateFrom := dateTo.Add(-1 * 10 * 24 * time.Hour)

	result := client.Tenants.UsageStatistics.ListSummaryAllTenants(ctx, usagestatistics.ListSummaryAllTenantsOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	summaries, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(summaries), 0, "At least 1 summary should be returned")
	assert.Greater(t, summaries[0].Get("deviceCount").Int(), int64(0), "Device count should be greater than 0")
}

func Test_GetCurrentTenant(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, client.Auth.Tenant, result.Data.Get("name").String())
	assert.NotEmpty(t, result.Data.Get("domainName").String(), "Domain name should not be empty")
}

func Test_GetTenantsWithNoSubtenants(t *testing.T) {
	t.Skip("Requires a multi tenant Cumulocity installation")
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Tenants.List(ctx, tenants.ListOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	tenantList, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tenantList), 0, "Should be an array")
}

func Test_CRUD_Tenant(t *testing.T) {
	// TODO: Test can't successfully execute because new tenants can't be deleted from a subtenant. It must be done from the management tenant
	t.SkipNow()
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	company := "redmilldesigns"
	domain := "mydomain." + client.BaseURL.Hostname()

	tenantInput := map[string]any{
		"company": company,
		"domain":  domain,
	}

	// Create tenant
	createResult := client.Tenants.Create(ctx, tenantInput)
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, company, createResult.Data.Get("company").String())
	assert.Equal(t, domain, createResult.Data.Get("domain").String())

	tenantID := createResult.Data.ID()

	// Get tenant
	getResult := client.Tenants.Get(ctx, tenantID)
	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, company, getResult.Data.Get("company").String())
	assert.Equal(t, domain, getResult.Data.Get("domain").String())

	// Update Tenant
	updateResult := client.Tenants.Update(ctx, tenantID, map[string]any{
		"contactName": "Homer",
		"status":      "SUSPENDED",
	})

	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, "Homer", updateResult.Data.Get("contactName").String())
	assert.Equal(t, "SUSPENDED", updateResult.Data.Get("status").String())

	// Delete tenant
	// TODO: Test needs to be executed from the management tenant
	/*
		deleteResult := client.Tenants.Delete(ctx, tenantID, tenants.DeleteOptions{})
		require.NoError(t, deleteResult.Err)
		assert.Equal(t, 204, deleteResult.HTTPStatus)
	*/
}

func Test_GetApplicationReferences(t *testing.T) {
	// TODO: Test needs to be executed from the management tenant
	t.SkipNow()
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Get current tenant
	currentTenantResult := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
	require.NoError(t, currentTenantResult.Err)
	assert.Equal(t, 200, currentTenantResult.HTTPStatus)

	tenantName := currentTenantResult.Data.Get("name").String()

	// Get application references for the tenant
	result := client.Tenants.ListApplicationReferences(ctx, tenantName, tenants.ListApplicationReferencesOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	references, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(references), 0, "Should have at least 1 application reference")
}

func Test_GetTenantLoginOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.LoginOptions.List(ctx, loginoptions.ListOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	options, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(options), 0, "Should have at least 1 login option")
}
