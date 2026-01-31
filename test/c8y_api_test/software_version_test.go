package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software/softwareversions"
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

	result := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateOptions{
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
	getResult := client.Repository.Software.Versions.Get(context.Background(), softwareversions.GetOptions{
		ID:          result.Data.ID(),
		WithParents: true,
	})
	require.NoError(t, getResult.Err)
	assert.Equal(t, result.Data.ID(), getResult.Data.ID())
	assert.Equal(t, "id", getResult.Meta["lookupMethod"])

	// Get by version name (lookup)
	getByVersionResult := client.Repository.Software.Versions.Get(context.Background(), softwareversions.GetOptions{
		SoftwareName: softwareName,
		Version:      version,
		WithParents:  true,
	})
	require.NoError(t, getByVersionResult.Err)
	assert.Equal(t, result.Data.ID(), getByVersionResult.Data.ID())
	assert.Equal(t, "version", getByVersionResult.Meta["lookupMethod"])
	assert.Equal(t, version, getByVersionResult.Meta["lookupVersion"])
	assert.Equal(t, softwareName, getByVersionResult.Meta["lookupSoftwareName"])

	// GetOrCreate again (should find existing)
	result2 := client.Repository.Software.Versions.GetOrCreateVersion(context.Background(), softwareversions.CreateOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      version,
	})
	require.NoError(t, result2.Err)
	assert.Equal(t, result.Data.ID(), result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool), "Should have found existing version")

	t.Cleanup(func() {
		// Delete the software version and its parent software item
		deleteResult := client.Repository.Software.Versions.Delete(context.Background(), softwareversions.DeleteOptions{
			SoftwareName: softwareName,
			Version:      version,
			ForceCascade: true,
		})
		assert.NoError(t, deleteResult.Err)
	})
}
