package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwarepatches"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FirmwareItems_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "fw-" + testingutils.RandomString(8)
	createRes := client.Repository.Firmware.Create(ctx, firmwareitems.CreateOptions{
		Name:        name,
		Description: "test firmware",
		DeviceType:  "c8y_Linux",
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Repository.Firmware.Delete(ctx, id, firmwareitems.DeleteOptions{})
	})

	getRes := client.Repository.Firmware.Get(ctx, id, firmwareitems.GetOptions{})
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.Repository.Firmware.Update(ctx, id, map[string]any{"description": "updated"})
	require.NoError(t, updRes.Err)

	listRes := client.Repository.Firmware.List(ctx, firmwareitems.ListOptions{Name: name})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Firmware.ListAll(ctx, firmwareitems.ListOptions{})
	require.NoError(t, itAll.Err())
}

func Test_FirmwareItems_GetOrCreate(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "fw-goc-" + testingutils.RandomString(8)
	res := client.Repository.Firmware.GetOrCreate(ctx, firmwareitems.CreateOptions{
		Name: name,
	})
	require.NoError(t, res.Err)
	id := res.Data.ID()
	t.Cleanup(func() {
		client.Repository.Firmware.Delete(ctx, id, firmwareitems.DeleteOptions{})
	})

	// Second call - should find existing
	res2 := client.Repository.Firmware.GetOrCreate(ctx, firmwareitems.CreateOptions{
		Name: name,
	})
	require.NoError(t, res2.Err)
	assert.Equal(t, id, res2.Data.ID())
}

func Test_FirmwareItems_Ref(t *testing.T) {
	ref := firmwareitems.NewRef()
	assert.Equal(t, "12345", ref.ByID("12345"))
	assert.Equal(t, "name:linux", ref.ByName("linux"))
	assert.Equal(t, "query:type eq 'c8y_Firmware'", ref.ByQuery("type eq 'c8y_Firmware'"))
}

func Test_FirmwareVersions_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create parent firmware item
	parent := client.Repository.Firmware.Create(ctx, firmwareitems.CreateOptions{
		Name: "fwParent-" + testingutils.RandomString(6),
	})
	require.NoError(t, parent.Err)
	parentID := parent.Data.ID()
	t.Cleanup(func() {
		client.Repository.Firmware.Delete(ctx, parentID, firmwareitems.DeleteOptions{
			ForceCascade: true,
		})
	})

	createRes := client.Repository.Firmware.Versions.Create(ctx, parentID, firmwareversions.CreateOptions{
		Version: "1.0.0",
		URL:     "https://example.com/firmware.bin",
	})
	require.NoError(t, createRes.Err)
	versionID := createRes.Data.ID()

	getRes := client.Repository.Firmware.Versions.Get(ctx, versionID, firmwareversions.GetOptions{})
	require.NoError(t, getRes.Err)

	updRes := client.Repository.Firmware.Versions.Update(ctx, versionID, map[string]any{"description": "updated"})
	require.NoError(t, updRes.Err)

	listRes := client.Repository.Firmware.Versions.List(ctx, firmwareversions.ListOptions{
		FirmwareID: parentID,
	})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Firmware.Versions.ListAll(ctx, firmwareversions.ListOptions{
		FirmwareID: parentID,
	})
	require.NoError(t, itAll.Err())

	delRes := client.Repository.Firmware.Versions.Delete(ctx, versionID, firmwareversions.DeleteOptions{})
	require.NoError(t, delRes.Err)
}

func Test_FirmwarePatches_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	parent := client.Repository.Firmware.Create(ctx, firmwareitems.CreateOptions{
		Name: "fwPatchParent-" + testingutils.RandomString(6),
	})
	require.NoError(t, parent.Err)
	parentID := parent.Data.ID()
	t.Cleanup(func() {
		client.Repository.Firmware.Delete(ctx, parentID, firmwareitems.DeleteOptions{
			ForceCascade: true,
		})
	})

	createRes := client.Repository.Firmware.Patches.Create(ctx, parentID, firmwarepatches.CreateOptions{
		Version:           "1.0.1",
		DependencyVersion: "1.0.0",
		URL:               "https://example.com/patch.bin",
	})
	require.NoError(t, createRes.Err)
	patchID := createRes.Data.ID()

	getRes := client.Repository.Firmware.Patches.Get(ctx, patchID, firmwarepatches.GetOptions{})
	require.NoError(t, getRes.Err)

	updRes := client.Repository.Firmware.Patches.Update(ctx, patchID, map[string]any{"description": "patched"})
	require.NoError(t, updRes.Err)

	listRes := client.Repository.Firmware.Patches.List(ctx, firmwarepatches.ListOptions{
		FirmwareID: parentID,
	})
	require.NoError(t, listRes.Err)

	itAll := client.Repository.Firmware.Patches.ListAll(ctx, firmwarepatches.ListOptions{
		FirmwareID: parentID,
	})
	require.NoError(t, itAll.Err())

	delRes := client.Repository.Firmware.Patches.Delete(ctx, patchID, firmwarepatches.DeleteOptions{})
	require.NoError(t, delRes.Err)
}
