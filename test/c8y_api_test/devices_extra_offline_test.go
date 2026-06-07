package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Devices_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	createRes := client.Devices.CreateDevice(ctx, "test-dev-"+t.Name())
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Devices.Delete(ctx, id, devices.DeleteOptions{})
	})

	getRes := client.Devices.Get(ctx, id, managedobjects.GetOptions{})
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.Devices.Update(ctx, id, map[string]any{"description": "hello"})
	require.NoError(t, updRes.Err)

	listAll := client.Devices.ListAll(ctx, devices.ListOptions{})
	require.NoError(t, listAll.Err())

	findRes := client.Devices.Find(ctx, devices.FindOptions{Type: "thin-edge.io"})
	require.NoError(t, findRes.Err)

	measRes := client.Devices.ListSupportedMeasurements(ctx, id)
	_ = measRes
	seriesRes := client.Devices.ListSupportedSeries(ctx, id)
	_ = seriesRes
}

func Test_Devices_GetOrCreateByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res := client.Devices.GetOrCreateByName(ctx, "myNewDev", "myType", map[string]any{"name": "myNewDev", "type": "myType"})
	require.NoError(t, res.Err)

	// Second call should fetch existing
	res2 := client.Devices.GetOrCreateByName(ctx, "myNewDev", "myType", map[string]any{"name": "myNewDev"})
	require.NoError(t, res2.Err)
	assert.Equal(t, res.Data.ID(), res2.Data.ID())
}

func Test_Devices_GetOrCreateByFragment(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res := client.Devices.GetOrCreateByFragment(ctx, "c8y_Test_Custom", map[string]any{
		"name":            "fragdev",
		"c8y_Test_Custom": map[string]any{},
	})
	require.NoError(t, res.Err)
}

func Test_Devices_GetOrCreateWith(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res := client.Devices.GetOrCreateWith(ctx, map[string]any{
		"name": "querydev",
	}, "name eq 'querydev'")
	require.NoError(t, res.Err)
}

func Test_ChildDevices_AssignUnassign(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	parent := testcore.CreateManagedObject(t, client)
	c1 := testcore.CreateManagedObject(t, client)

	assignRes := client.Devices.ChildDevices.Assign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, assignRes.Err)

	listRes := client.Devices.ChildDevices.List(ctx, parent.Data.ID(), childdevices.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.Devices.ChildDevices.ListAll(ctx, parent.Data.ID(), childdevices.ListOptions{})
	require.NoError(t, itAll.Err())

	getRes := client.Devices.ChildDevices.Get(ctx, parent.Data.ID(), c1.Data.ID())
	_ = getRes

	createRes := client.Devices.ChildDevices.Create(ctx, parent.Data.ID(), map[string]any{
		"name": "subdev",
	})
	require.NoError(t, createRes.Err)

	unassignRes := client.Devices.ChildDevices.Unassign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, unassignRes.Err)
}

func Test_ChildAssets_AssignUnassign(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	parent := testcore.CreateManagedObject(t, client)
	c1 := testcore.CreateManagedObject(t, client)

	assignRes := client.Devices.ChildAssets.Assign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, assignRes.Err)

	listRes := client.Devices.ChildAssets.List(ctx, parent.Data.ID(), childassets.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.Devices.ChildAssets.ListAll(ctx, parent.Data.ID(), childassets.ListOptions{})
	require.NoError(t, itAll.Err())

	getRes := client.Devices.ChildAssets.Get(ctx, parent.Data.ID(), c1.Data.ID())
	_ = getRes

	createRes := client.Devices.ChildAssets.Create(ctx, parent.Data.ID(), map[string]any{
		"name": "subasset",
	})
	require.NoError(t, createRes.Err)

	unassignRes := client.Devices.ChildAssets.Unassign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, unassignRes.Err)
}

func Test_ChildAdditions_AssignUnassign(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	parent := testcore.CreateManagedObject(t, client)
	c1 := testcore.CreateManagedObject(t, client)

	assignRes := client.Devices.ChildAdditions.Assign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, assignRes.Err)

	listRes := client.Devices.ChildAdditions.List(ctx, parent.Data.ID(), childadditions.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.Devices.ChildAdditions.ListAll(ctx, parent.Data.ID(), childadditions.ListOptions{})
	require.NoError(t, itAll.Err())

	getRes := client.Devices.ChildAdditions.Get(ctx, parent.Data.ID(), c1.Data.ID())
	_ = getRes

	createRes := client.Devices.ChildAdditions.Create(ctx, parent.Data.ID(), map[string]any{
		"name": "subaddition",
	})
	require.NoError(t, createRes.Err)

	unassignRes := client.Devices.ChildAdditions.Unassign(ctx, parent.Data.ID(), c1.Data.ID())
	require.NoError(t, unassignRes.Err)
}
