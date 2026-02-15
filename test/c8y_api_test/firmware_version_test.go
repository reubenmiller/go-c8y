package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FirmwareVersion(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	firmwareName := "firmware" + testingutils.RandomString(16)
	version := "1.0.1"
	tempFile := testingutils.CreateTempFile(t, "test firmware binary content")

	result := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      version,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, version, result.Data.Version())
	assert.NotEmpty(t, result.Data.ID())
	assert.False(t, result.Meta["found"].(bool), "Should have created new version")

	getResult := client.Repository.Firmware.Versions.Get(
		context.Background(),
		result.Data.ID(),
		firmwareversions.GetOptions{
			WithParents: true,
		},
	)
	require.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	result2 := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      version,
	})
	require.NoError(t, result2.Err)
	assert.Equal(t, result.Data.ID(), result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool), "Should have found existing version")

	t.Cleanup(func() {
		deleteResult := client.Repository.Firmware.Versions.Delete(
			context.Background(),
			firmwareversions.NewRef().ByVersionAndName(version, firmwareName),
			firmwareversions.DeleteOptions{
				ForceCascade: true,
			})
		assert.NoError(t, deleteResult.Err)
	})
}

func Test_FirmwareVersionResolver_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	version := "1.0.0"
	tempFile := testingutils.CreateTempFile(t, "test firmware binary content")

	result := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      version,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Versions.Delete(context.Background(), result.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	getResult := client.Repository.Firmware.Versions.Get(
		context.Background(),
		result.Data.ID(),
		firmwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	getResult2 := client.Repository.Firmware.Versions.Get(
		context.Background(),
		firmwareversions.NewRef().ByID(result.Data.ID()),
		firmwareversions.GetOptions{},
	)
	assert.NoError(t, getResult2.Err)
	assert.Equal(t, result.Data.ID(), getResult2.Data.ID())
}

func Test_FirmwareVersionResolver_ByVersionAndName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	version := "3.0.0"
	tempFile := testingutils.CreateTempFile(t, "test firmware binary content")

	result := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      version,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Versions.Delete(context.Background(), result.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	getResult := client.Repository.Firmware.Versions.Get(
		context.Background(),
		firmwareversions.NewRef().ByVersionAndName(version, firmwareName),
		firmwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "version", getResult.Meta["resolverType"])
	assert.Equal(t, version, getResult.Meta["version"])
	assert.Equal(t, firmwareName, getResult.Meta["firmwareName"])
}

func Test_FirmwareVersionDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	version := "6.0.0"
	tempFile := testingutils.CreateTempFile(t, "test firmware binary content")

	result := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      version,
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Firmware.Versions.Delete(context.Background(), result.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	t.Run("Get deferred by version and name", func(t *testing.T) {
		deferredResult := client.Repository.Firmware.Versions.Get(
			api.WithDeferredExecution(context.Background(), true),
			firmwareversions.NewRef().ByVersionAndName(version, firmwareName),
			firmwareversions.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, result.Data.ID(), deferredResult.Meta["id"])

		execResult := deferredResult.Execute(context.Background())
		assert.NoError(t, execResult.Err)
		assert.Equal(t, result.Data.ID(), execResult.Data.ID())
	})

	t.Run("Get deferred by version and firmware ID", func(t *testing.T) {
		firmwareResult := client.Repository.Firmware.Get(
			context.Background(),
			firmwareitems.NewRef().ByName(firmwareName),
			firmwareitems.GetOptions{},
		)
		require.NoError(t, firmwareResult.Err)
		firmwareID := firmwareResult.Data.ID()

		deferredResult := client.Repository.Firmware.Versions.Get(
			api.WithDeferredExecution(context.Background(), true),
			firmwareversions.NewRef().ByVersion(version, firmwareID),
			firmwareversions.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, result.Data.ID(), deferredResult.Meta["id"])
	})
}

func Test_FirmwareVersionList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	firmwareName := "firmware" + testingutils.RandomString(16)
	tempFile := testingutils.CreateTempFile(t, "test firmware binary content")

	v1 := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      "1.0.0",
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, v1.Err)

	v2 := client.Repository.Firmware.Versions.GetOrCreateVersion(context.Background(), firmwareversions.CreateVersionOptions{
		FirmwareName: firmwareName,
		Version:      "2.0.0",
		File: firmwareversions.UploadFileOptions{
			Name:        "firmware.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, v2.Err)

	t.Cleanup(func() {
		client.Repository.Firmware.Versions.Delete(context.Background(), v1.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
		client.Repository.Firmware.Versions.Delete(context.Background(), v2.Data.ID(), firmwareversions.DeleteOptions{ForceCascade: true})
	})

	firmwareResult := client.Repository.Firmware.Get(
		context.Background(),
		firmwareitems.NewRef().ByName(firmwareName),
		firmwareitems.GetOptions{},
	)
	require.NoError(t, firmwareResult.Err)

	listResult := client.Repository.Firmware.Versions.List(context.Background(), firmwareversions.ListOptions{
		FirmwareID: firmwareResult.Data.ID(),
	})
	assert.NoError(t, listResult.Err)

	count := 0
	for range listResult.Data.Iter() {
		count++
	}
	assert.GreaterOrEqual(t, count, 2, "Should have at least 2 versions")
}
