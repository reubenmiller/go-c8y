package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/configuration"
	softwareitems "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	softwareversions "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Configuration_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "cfg-" + testingutils.RandomString(8)
	createRes := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "agent.config",
		Description:       "test config",
		URL:               "https://example.com/cfg.txt",
		DeviceType:        "c8y_Linux",
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, id, configuration.DeleteOptions{})
	})

	getRes := client.Repository.Configuration.Get(ctx, id, configuration.GetOptions{})
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.Repository.Configuration.Update(ctx, id, map[string]any{"description": "updated"})
	require.NoError(t, updRes.Err)

	listRes := client.Repository.Configuration.List(ctx, configuration.ListOptions{Name: name})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Configuration.ListAll(ctx, configuration.ListOptions{})
	require.NoError(t, itAll.Err())
}

func Test_Software_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "sw-" + testingutils.RandomString(8)
	createRes := client.Repository.Software.Create(ctx, map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"description":  "test software",
		"softwareType": "rpm",
		"c8y_Filter":   map[string]any{"type": "c8y_Linux"},
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Repository.Software.Delete(ctx, id, softwareitems.DeleteOptions{})
	})

	getRes := client.Repository.Software.Get(ctx, id, softwareitems.GetOptions{})
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.Repository.Software.Update(ctx, id, map[string]any{"description": "updated"})
	require.NoError(t, updRes.Err)

	listRes := client.Repository.Software.List(ctx, softwareitems.ListOptions{Name: name})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Software.ListAll(ctx, softwareitems.ListOptions{})
	require.NoError(t, itAll.Err())
}

func Test_SoftwareVersions_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	parent := client.Repository.Software.Create(ctx, map[string]any{
		"name": "swParent-" + testingutils.RandomString(6),
		"type": "c8y_Software",
	})
	require.NoError(t, parent.Err)
	parentID := parent.Data.ID()
	t.Cleanup(func() {
		client.Repository.Software.Delete(ctx, parentID, softwareitems.DeleteOptions{})
	})

	createRes := client.Repository.Software.Versions.Create(ctx, parentID, softwareversions.CreateOptions{
		Version: "1.0.0",
		URL:     "https://example.com/sw.bin",
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()

	listRes := client.Repository.Software.Versions.List(ctx, softwareversions.ListOptions{
		SoftwareID: parentID,
	})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Software.Versions.ListAll(ctx, softwareversions.ListOptions{
		SoftwareID: parentID,
	})
	require.NoError(t, itAll.Err())

	updRes := client.Repository.Software.Versions.Update(ctx, id, map[string]any{"description": "updated"})
	require.NoError(t, updRes.Err)

	delRes := client.Repository.Software.Versions.Delete(ctx, id, softwareversions.DeleteOptions{})
	require.NoError(t, delRes.Err)
}
