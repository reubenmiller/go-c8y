package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/devicestatistics"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Monthly device statistics
// ---------------------------------------------------------------------------

func Test_DeviceStatistics_Monthly_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DeviceStatistics.ListMonthly(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-01",
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DeviceStatistics_Monthly_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DeviceStatistics.ListMonthly(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-01",
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/tenant/statistics/device/t123/monthly/2024-01-01")

	// Execute with dry run to confirm full round-trip
	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DeviceStatistics_Monthly_DeviceIDFilter(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DeviceStatistics.ListMonthly(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-01",
		DeviceID: "12345",
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.Path, "/tenant/statistics/device/t123/monthly/2024-01-01")
	assert.Contains(t, prepared.Request.URL.RawQuery, "deviceId=12345")
}

func Test_DeviceStatistics_Monthly_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-01",
	}
	opts.PageSize = 10
	opts.WithTotalPages = true

	prepared := client.DeviceStatistics.ListMonthly(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=10")
	assert.Contains(t, prepared.Request.URL.RawQuery, "withTotalPages=true")
}

// ---------------------------------------------------------------------------
// Daily device statistics
// ---------------------------------------------------------------------------

func Test_DeviceStatistics_Daily_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DeviceStatistics.ListDaily(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-15",
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DeviceStatistics_Daily_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DeviceStatistics.ListDaily(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-15",
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/tenant/statistics/device/t123/daily/2024-01-15")

	// Execute with dry run
	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DeviceStatistics_Daily_DeviceIDFilter(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DeviceStatistics.ListDaily(ctx, devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-15",
		DeviceID: "67890",
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.Path, "/tenant/statistics/device/t123/daily/2024-01-15")
	assert.Contains(t, prepared.Request.URL.RawQuery, "deviceId=67890")
}

func Test_DeviceStatistics_Daily_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := devicestatistics.ListOptions{
		TenantID: "t123",
		Date:     "2024-01-15",
	}
	opts.PageSize = 20

	prepared := client.DeviceStatistics.ListDaily(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=20")
}
