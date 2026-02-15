package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TenantsList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	item := client.Tenants.List(context.Background(), tenants.ListOptions{})
	assert.NoError(t, item.Err)
	assert.NotEmpty(t, item.Meta["self"])
}

func Test_GetCurrent(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// Get Current
	currentTenant := client.Tenants.Current.Get(context.Background(), currenttenant.GetOptions{})
	assert.NoError(t, currentTenant.Err)
	assert.NotEmpty(t, currentTenant.Data.Name())
}
