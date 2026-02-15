package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SoftwareVersion(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// Get Or Create software version (new item)
	softwareName := "software" + testingutils.RandomString(16)
	version := "1.0.1"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "dummy.txt",
			ContentType: "text/plain",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, version, result.Data.Version())
	assert.NotEmpty(t, result.Data.ID())
	assert.False(t, result.Meta["found"].(bool), "Should have created new version")

	// Get by ID
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		result.Data.ID(),
		softwareversions.GetOptions{
			WithParents: true,
		},
	)
	require.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	// Get by version name (lookup)
	getByVersionDeferred := client.Repository.Software.Versions.Get(
		api.WithDeferredExecution(context.Background(), true),
		softwareversions.NewRef().ByVersionAndName(version, softwareName),
		softwareversions.GetOptions{
			WithParents: true,
		},
	)
	require.NoError(t, getByVersionDeferred.Err)
	assert.True(t, getByVersionDeferred.IsDeferred())
	assert.NotEmpty(t, getByVersionDeferred.Meta["id"])

	// execute
	getByVersionResult := getByVersionDeferred.Execute(context.Background())
	assert.Equal(t, result.Data.ID(), getByVersionResult.Data.ID())
	assert.Equal(t, "version", getByVersionResult.Meta["resolverType"])
	assert.Equal(t, version, getByVersionResult.Meta["version"])
	assert.Equal(t, softwareName, getByVersionResult.Meta["softwareName"])

	// GetOrCreate again (should find existing)
	result2 := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
	})
	require.NoError(t, result2.Err)
	assert.Equal(t, result.Data.ID(), result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool), "Should have found existing version")

	t.Cleanup(func() {
		// Delete the software version and its parent software item
		deleteResult := client.Repository.Software.Versions.Delete(
			context.Background(),
			softwareversions.NewRef().ByVersionAndName(version, softwareName),
			softwareversions.DeleteOptions{
				ForceCascade: true,
			})
		assert.NoError(t, deleteResult.Err)
	})
}

func Test_SoftwareVersionResolver_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "1.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	// Test direct ID resolution
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		result.Data.ID(),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["resolverType"])

	// Test Ref.ByID
	getResult2 := client.Repository.Software.Versions.Get(
		context.Background(),
		softwareversions.NewRef().ByID(result.Data.ID()),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult2.Err)
	assert.Equal(t, result.Data.ID(), getResult2.Data.ID())
}

func Test_SoftwareVersionResolver_ByVersion(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "2.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	// Get software item to retrieve its ID
	softwareResult := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName, softwareType),
		softwareitems.GetOptions{},
	)
	require.NoError(t, softwareResult.Err)
	softwareID := softwareResult.Data.ID()

	// Test version + software ID resolution
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		softwareversions.NewRef().ByVersion(version, softwareID),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "version", getResult.Meta["resolverType"])
	assert.Equal(t, version, getResult.Meta["version"])
	assert.Equal(t, softwareID, getResult.Meta["softwareID"])
}

func Test_SoftwareVersionResolver_ByVersionAndName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "3.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	// Test version + name resolution (without type)
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		softwareversions.NewRef().ByVersionAndName(version, softwareName),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "version", getResult.Meta["resolverType"])
	assert.Equal(t, version, getResult.Meta["version"])
	assert.Equal(t, softwareName, getResult.Meta["softwareName"])

	// Test version + name + type resolution
	getResult2 := client.Repository.Software.Versions.Get(
		context.Background(),
		softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult2.Err)
	assert.Equal(t, result.Data.ID(), getResult2.Data.ID())
	assert.Equal(t, "version", getResult2.Meta["resolverType"])
	assert.Equal(t, version, getResult2.Meta["version"])
	assert.Equal(t, softwareName, getResult2.Meta["softwareName"])
	assert.Equal(t, softwareType, getResult2.Meta["softwareType"])
}

func Test_SoftwareVersionResolver_Errors(t *testing.T) {
	client := testcore.CreateTestClient(t)

	t.Run("empty identifier", func(t *testing.T) {
		result := client.Repository.Software.Versions.Get(
			context.Background(),
			"",
			softwareversions.GetOptions{},
		)
		assert.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "cannot be empty")
	})

	t.Run("version not found", func(t *testing.T) {
		result := client.Repository.Software.Versions.Get(
			context.Background(),
			softwareversions.NewRef().ByVersionAndName("99.99.99", "nonexistent-"+testingutils.RandomString(16)),
			softwareversions.GetOptions{},
		)
		assert.Error(t, result.Err)
	})

	t.Run("software not found when resolving version by name", func(t *testing.T) {
		result := client.Repository.Software.Versions.Get(
			context.Background(),
			softwareversions.NewRef().ByVersionAndName("1.0.0", "definitely-does-not-exist-"+testingutils.RandomString(16)),
			softwareversions.GetOptions{},
		)
		assert.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "not found")
	})
}

func Test_SoftwareVersionUpdate_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "4.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	// Update by version and name
	updateResult := client.Repository.Software.Versions.Update(
		context.Background(),
		softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
		map[string]any{
			"c8y_Firmware": map[string]any{
				"url": "https://example.com/updated.bin",
			},
		},
	)
	assert.NoError(t, updateResult.Err)
	assert.Equal(t, "version", updateResult.Meta["resolverType"])
}

func Test_SoftwareVersionDelete_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "5.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)

	// Delete by version and name
	deleteResult := client.Repository.Software.Versions.Delete(
		context.Background(),
		softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
		softwareversions.DeleteOptions{ForceCascade: true},
	)
	assert.NoError(t, deleteResult.Err)
	assert.Equal(t, "version", deleteResult.Meta["resolverType"])

	// Verify deletion
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		result.Data.ID(),
		softwareversions.GetOptions{},
	)
	assert.Error(t, getResult.Err)
}

func Test_SoftwareVersionDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "6.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	t.Run("Get deferred by version and name", func(t *testing.T) {
		deferredResult := client.Repository.Software.Versions.Get(
			api.WithDeferredExecution(context.Background(), true),
			softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
			softwareversions.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, result.Data.ID(), deferredResult.Meta["id"])

		// Execute the deferred operation
		execResult := deferredResult.Execute(context.Background())
		assert.NoError(t, execResult.Err)
		assert.Equal(t, result.Data.ID(), execResult.Data.ID())
	})

	t.Run("Get deferred by version and software ID", func(t *testing.T) {
		// Get software item to retrieve its ID
		softwareResult := client.Repository.Software.Get(
			context.Background(),
			softwareitems.NewRef().ByName(softwareName, softwareType),
			softwareitems.GetOptions{},
		)
		require.NoError(t, softwareResult.Err)
		softwareID := softwareResult.Data.ID()

		deferredResult := client.Repository.Software.Versions.Get(
			api.WithDeferredExecution(context.Background(), true),
			softwareversions.NewRef().ByVersion(version, softwareID),
			softwareversions.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, result.Data.ID(), deferredResult.Meta["id"])
	})

	t.Run("Update deferred", func(t *testing.T) {
		deferredResult := client.Repository.Software.Versions.Update(
			api.WithDeferredExecution(context.Background(), true),
			softwareversions.NewRef().ByVersionAndName(version, softwareName),
			map[string]any{"description": "Deferred update"},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
	})

	t.Run("Delete deferred", func(t *testing.T) {
		deferredResult := client.Repository.Software.Versions.Delete(
			api.WithDeferredExecution(context.Background(), true),
			softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
			softwareversions.DeleteOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
	})
}

func Test_SoftwareVersionMetadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	version := "7.0.0"
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create software version
	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
		File: softwareversions.UploadFileOptions{
			Name:        "test.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})

	// Get with version resolver - check metadata
	getResult := client.Repository.Software.Versions.Get(
		context.Background(),
		softwareversions.NewRef().ByVersionAndName(version, softwareName, softwareType),
		softwareversions.GetOptions{},
	)
	assert.NoError(t, getResult.Err)
	assert.Equal(t, "version", getResult.Meta["resolverType"])
	assert.Equal(t, version, getResult.Meta["version"])
	assert.Equal(t, softwareName, getResult.Meta["softwareName"])
	assert.Equal(t, softwareType, getResult.Meta["softwareType"])
	assert.Equal(t, result.Data.ID(), getResult.Meta["id"])
	assert.Equal(t, "version:"+version+":name:"+softwareName+":"+softwareType, getResult.Meta["identifier"])
}

func Test_SoftwareVersionGetOrCreate_MultipleVersions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"
	tempFile := testingutils.CreateTempFile(t, "test binary content")

	// Create first version
	v1Result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      "1.0.0",
		File: softwareversions.UploadFileOptions{
			Name:        "v1.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, v1Result.Err)
	assert.False(t, v1Result.Meta["found"].(bool))

	// Create second version (same software, different version)
	v2Result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      "2.0.0",
		File: softwareversions.UploadFileOptions{
			Name:        "v2.bin",
			ContentType: "application/octet-stream",
			FilePath:    tempFile,
		},
	})
	require.NoError(t, v2Result.Err)
	assert.False(t, v2Result.Meta["found"].(bool))
	assert.NotEqual(t, v1Result.Data.ID(), v2Result.Data.ID(), "Different versions should have different IDs")
	// Both versions should belong to the same software item (verified by name)

	// GetOrCreate v1 again (should find existing)
	v1AgainResult := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      "1.0.0",
	})
	require.NoError(t, v1AgainResult.Err)
	assert.True(t, v1AgainResult.Meta["found"].(bool))
	assert.Equal(t, v1Result.Data.ID(), v1AgainResult.Data.ID())

	t.Cleanup(func() {
		client.Repository.Software.Versions.Delete(context.Background(), v1Result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: false})
		client.Repository.Software.Versions.Delete(context.Background(), v2Result.Data.ID(), softwareversions.DeleteOptions{ForceCascade: true})
	})
}
