package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestTenantService_GetTenantStatisticsSummary(t *testing.T) {
	client := createTestClient()

	dateFrom, dateTo := c8y.GetDateRange("10d")

	summary, resp, err := client.Tenant.GetTenantStatisticsSummary(
		context.Background(),
		&c8y.TenantSummaryOptions{
			DateFrom: dateFrom,
			DateTo:   dateTo,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, summary.StorageSize > 0, "Storage size should be greater than 0")
	testingutils.Assert(t, summary.RequestCount > 0, "Request count should be greater than 0")
}

func TestTenantService_GetTenantStatistics(t *testing.T) {
	client := createTestClient()

	dateFrom, dateTo := c8y.GetDateRange("10d")

	statistics, resp, err := client.Tenant.GetTenantStatistics(
		context.Background(),
		&c8y.TenantStatisticsOptions{
			DateFrom:          dateFrom,
			DateTo:            dateTo,
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 10, len(statistics.UsageStatistics))
}

func TestTenantService_GetAllTenantsStatisticsSummary(t *testing.T) {
	// TODO: Test requires a tenant with subtenant capabilities
	t.SkipNow()
	client := createTestClient()

	dateFrom, dateTo := c8y.GetDateRange("10d")

	summaries, resp, err := client.Tenant.GetAllTenantsStatisticsSummary(
		context.Background(),
		&c8y.TenantStatisticsOptions{
			DateFrom: dateFrom,
			DateTo:   dateTo,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(summaries) > 0, "At least 1 summary should be returned")
	testingutils.Assert(t, summaries[0].DeviceCount > 0, "Request count should be greater than 0")
}

func TestTenantService_GetCurrentTenant(t *testing.T) {
	client := createTestClient()
	tenant, resp, err := client.Tenant.GetCurrentTenant(context.Background())

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, client.TenantName, tenant.Name)
	testingutils.Assert(t, tenant.DomainName != "", "Domain name should not be empty")
}

func TestTenantService_GetTenantsWithNoSubtenants(t *testing.T) {
	client := createTestClient()
	tenantCollection, resp, err := client.Tenant.GetTenants(
		context.Background(),
		c8y.NewPaginationOptions(100),
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(tenantCollection.Tenants) >= 0, "Should be an array")
}

func TestTenantService_CRUDTenant(t *testing.T) {
	// TODO: Test can't successfully execute because new tenants can't be deleted from a subtenant. It must be done from the management tenant
	t.SkipNow()
	client := createTestClient()

	company := "redmilldesigns"
	domain := fmt.Sprintf("%s.%s", "mydomain", client.BaseURL.Hostname())
	tenantInput := c8y.NewTenant(company, domain)

	//
	// Create tenant
	tenant, resp, err := client.Tenant.Create(
		context.Background(),
		tenantInput,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, company, tenant.Company)
	testingutils.Equals(t, domain, tenant.Domain)

	//
	// Get tenant
	tenantByID, resp, err := client.Tenant.GetTenant(
		context.Background(),
		tenant.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, company, tenantByID.Company)
	testingutils.Equals(t, domain, tenantByID.Domain)

	//
	// Update Tenant
	updatedTenant, resp, err := client.Tenant.Update(
		context.Background(),
		tenant.ID,
		&c8y.Tenant{
			ContactName: "Homer",
			Status:      "SUSPENDED",
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "Homer", updatedTenant.ContactName)
	testingutils.Equals(t, "SUSPENDED", updatedTenant.Status)

	//
	// Delete tenant
	// TODO: Test needs to be executed from the management tenant
	/*
		resp, err = client.Tenant.Delete(
			context.Background(),
			tenant.ID,
		)

		testingutils.Ok(t, err)
		testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)
	*/
}

func TestTenantService_GetApplicationReferences(t *testing.T) {
	// TODO: Test needs to be executed from the management tenant
	t.SkipNow()
	client := createTestClient()
	currentTenant, resp, err := client.Tenant.GetCurrentTenant(
		context.Background(),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)

	appReferenceCollection, resp, err := client.Tenant.GetApplicationReferences(
		context.Background(),
		currentTenant.Name,
		c8y.NewPaginationOptions(100),
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(appReferenceCollection.References) > 0, "Should have at least 1 application reference")
}
