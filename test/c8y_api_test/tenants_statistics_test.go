package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TenantsStatisticsList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	usage := client.Tenants.UsageStatistics.List(context.Background(), usagestatistics.ListOptions{})
	assert.NoError(t, usage.Err)
	assert.Greater(t, usage.Data.Length(), 0)
}

func Test_TenantsStatisticsListSummary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	usage := client.Tenants.UsageStatistics.ListSummary(context.Background(), usagestatistics.ListSummaryOptions{
		DateFrom: time.Now().AddDate(-1, 0, 0),
	})
	assert.NoError(t, usage.Err)
	assert.Greater(t, usage.Data.Length(), 0)
}
func Test_TenantsStatisticsListSummaryAllTenants(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	usage := client.Tenants.UsageStatistics.ListSummaryAllTenants(context.Background(), usagestatistics.ListSummaryAllTenantsOptions{
		DateFrom: time.Now().AddDate(-1, 0, 0),
	})
	assert.NoError(t, usage.Err)
	assert.Greater(t, usage.Data.Length(), 0)
}
