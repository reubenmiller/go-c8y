package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/firmware/firmwarepatches"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FirmwarePatch(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	firmwareName := "firmware" + testingutils.RandomString(16)
	baseVersion := "1.0.0"
	patchVersion := "1.0.1"

	tempFile := testingutils.CreateTempFile(t, "test patch binary content")

	baseResult := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,

		Version: baseVersion,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, baseResult.Err)

	result := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           patchVersion,
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, patchVersion, result.Data.Version())
	assert.Equal(t, baseVersion, result.Data.Dependency())
	assert.NotEmpty(t, result.Data.ID())
	assert.False(t, result.Meta["found"].(bool), "Should have created new patch")

	getResult := client.Repository.Firmware.Patches.Get(
		context.Background(),
		result.Data.ID(),
		firmwarepatches.GetOptions{
			WithParents: true,
		},
	)
	require.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	result2 := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           patchVersion,
		DependencyVersion: baseVersion,
	})
	require.NoError(t, result2.Err)
	assert.Equal(t, result.Data.ID(), result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool), "Should have found existing patch")

	t.Cleanup(func() {
		deleteResult := client.Repository.Firmware.Patches.Delete(
			context.Background(),
			result.Data.ID(),
			firmwarepatches.DeleteOptions{
				ForceCascade: true,
			})
		assert.NoError(t, deleteResult.Err)

		client.Repository.Firmware.Versions.Delete(
			context.Background(),
			baseResult.Data.ID(),
			firmwareversions.DeleteOptions{ForceCascade: true})
	})
}

func Test_FirmwarePatchResolver_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	baseVersion := "1.0.0"
	patchVersion := "1.0.1"

	tempFile := testingutils.CreateTempFile(t, "test patch binary content")

	baseResult := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,

		Version: baseVersion,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, baseResult.Err)

	result := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           patchVersion,
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Patches.Delete(context.Background(), result.Data.ID(), firmwarepatches.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Versions.Delete(context.Background(), baseResult.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	getResult := client.Repository.Firmware.Patches.Get(
		context.Background(),
		result.Data.ID(),
		firmwarepatches.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	getResult2 := client.Repository.Firmware.Patches.Get(
		context.Background(),
		firmwarepatches.NewRef().ByID(result.Data.ID()),
		firmwarepatches.GetOptions{},
	)
	assert.NoError(t, getResult2.Err)
	assert.Equal(t, result.Data.ID(), getResult2.Data.ID())
}

func Test_FirmwarePatchResolver_ByVersionAndDependency(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	baseVersion := "2.0.0"
	patchVersion := "2.0.1"

	tempFile := testingutils.CreateTempFile(t, "test patch binary content")

	baseResult := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,

		Version: baseVersion,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, baseResult.Err)

	result := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           patchVersion,
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Patches.Delete(context.Background(), result.Data.ID(), firmwarepatches.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Versions.Delete(context.Background(), baseResult.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	firmwareResult := client.Repository.Firmware.Get(
		context.Background(),
		firmwareitems.NewRef().ByName(firmwareName),
		firmwareitems.GetOptions{},
	)
	require.NoError(t, firmwareResult.Err)
	firmwareID := firmwareResult.Data.ID()

	getResult := client.Repository.Firmware.Patches.Get(
		context.Background(),
		firmwarepatches.NewRef().ByVersionAndDependency(patchVersion, baseVersion, firmwareID),
		firmwarepatches.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "version", getResult.Meta["resolverType"])
	assert.Equal(t, patchVersion, getResult.Meta["version"])
	assert.Equal(t, baseVersion, getResult.Meta["dependency"])
	assert.Equal(t, firmwareID, getResult.Meta["firmwareID"])
}

func Test_FirmwarePatchDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	baseVersion := "5.0.0"
	patchVersion := "5.0.1"

	tempFile := testingutils.CreateTempFile(t, "test patch binary content")

	baseResult := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,

		Version: baseVersion,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, baseResult.Err)

	result := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           patchVersion,
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Patches.Delete(context.Background(), result.Data.ID(), firmwarepatches.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Versions.Delete(context.Background(), baseResult.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	// Get firmware ID for resolver
	firmwareResult := client.Repository.Firmware.Get(
		context.Background(),
		firmwareitems.NewRef().ByName(firmwareName),
		firmwareitems.GetOptions{},
	)
	require.NoError(t, firmwareResult.Err)
	firmwareID := firmwareResult.Data.ID()

	t.Run("Get deferred by version, dependency and firmware ID", func(t *testing.T) {
		deferredResult := client.Repository.Firmware.Patches.Get(
			c8y_api.WithDeferredExecution(context.Background(), true),
			firmwarepatches.NewRef().ByVersionAndDependency(patchVersion, baseVersion, firmwareID),
			firmwarepatches.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, result.Data.ID(), deferredResult.Meta["id"])

		execResult := deferredResult.Execute(context.Background())
		assert.NoError(t, execResult.Err)
		assert.Equal(t, result.Data.ID(), execResult.Data.ID())
	})
}

func Test_FirmwarePatchList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	baseVersion := "6.0.0"

	tempFile := testingutils.CreateTempFile(t, "test patch binary content")

	baseResult := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,

		Version: baseVersion,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, baseResult.Err)

	p1 := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           "6.0.1",
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch1.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, p1.Err)

	p2 := client.Repository.Firmware.Patches.GetOrCreatePatch(context.Background(), firmwarepatches.CreatePatchOptions{
		FirmwareName:      firmwareName,
		Version:           "6.0.2",
		DependencyVersion: baseVersion,
		File: core.UploadFileOptions{
			Name:        "patch2.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, p2.Err)

	t.Cleanup(func() {
		client.Repository.Firmware.Patches.Delete(context.Background(), p1.Data.ID(), firmwarepatches.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Patches.Delete(context.Background(), p2.Data.ID(), firmwarepatches.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Versions.Delete(context.Background(), baseResult.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	firmwareResult := client.Repository.Firmware.Get(
		context.Background(),
		firmwareitems.NewRef().ByName(firmwareName),
		firmwareitems.GetOptions{},
	)
	require.NoError(t, firmwareResult.Err)

	listResult := client.Repository.Firmware.Patches.List(context.Background(), firmwarepatches.ListOptions{
		FirmwareID: firmwareResult.Data.ID(),
	})
	assert.NoError(t, listResult.Err)

	count := 0
	for range listResult.Data.Iter() {
		count++
	}
	assert.GreaterOrEqual(t, count, 2, "Should have at least 2 patches")
}
