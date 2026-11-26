package usagestatisticsfiles_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/usagestatisticsfiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The service is reachable as Client.Tenants.UsageStatisticsFiles.
func TestUsageStatisticsFiles_WiredIntoClient(t *testing.T) {
	client, stop := examplesetup.New()
	defer stop()
	require.NotNil(t, client.Tenants.UsageStatisticsFiles)

	var got *http.Request
	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithDryRunHandler(ctx, func(r *http.Request) { got = r })

	require.NoError(t, client.Tenants.UsageStatisticsFiles.List(ctx, usagestatisticsfiles.ListOptions{}).Err)
	require.NotNil(t, got)
	assert.Equal(t, "/tenant/statistics/files", got.URL.Path)
}
