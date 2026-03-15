package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_SoftwareVersionUpsertByVersion_Create(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a software item first
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "application"

	softwareResult := client.Repository.Software.GetOrCreateByName(ctx, softwareName, softwareType, map[string]any{
		"name":         softwareName,
		"type":         "c8y_Software",
		"softwareType": softwareType,
	})
	assert.NoError(t, softwareResult.Err)
	softwareID := softwareResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Software.Delete(ctx, softwareID, softwareitems.DeleteOptions{})
	})

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-binary-v1.bin")
	err := os.WriteFile(testFile, []byte("test content v1"), 0644)
	assert.NoError(t, err)

	// First upsert should create
	opt := softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    "1.0.0",
		File: softwareversions.UploadFileOptions{
			FilePath: testFile,
			Name:     "test-binary-v1.bin",
		},
	}

	result1 := client.Repository.Software.Versions.UpsertByVersion(ctx, opt)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.False(t, result1.Meta["found"].(bool))

	versionID := result1.Data.ID()

	// Second upsert with same version should update
	testFile2 := filepath.Join(tempDir, "test-binary-v2.bin")
	err = os.WriteFile(testFile2, []byte("test content v2 - updated"), 0644)
	assert.NoError(t, err)

	opt.File.FilePath = testFile2
	opt.File.Name = "test-binary-v2.bin"

	result2 := client.Repository.Software.Versions.UpsertByVersion(ctx, opt)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.True(t, result2.Meta["found"].(bool))

	versionID2 := result2.Data.ID()
	assert.Equal(t, versionID, versionID2, "Should update same version")

	// Cleanup
	_ = client.Repository.Software.Versions.Delete(ctx, versionID, softwareversions.DeleteOptions{})
}

func Test_SoftwareVersionUpsertByVersion_WithSoftwareName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a software item first
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "firmware"

	softwareResult := client.Repository.Software.GetOrCreateByName(ctx, softwareName, softwareType, map[string]any{
		"name":         softwareName,
		"type":         "c8y_Software",
		"softwareType": softwareType,
	})
	assert.NoError(t, softwareResult.Err)
	softwareID := softwareResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Software.Delete(ctx, softwareID, softwareitems.DeleteOptions{})
	})

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "firmware.bin")
	err := os.WriteFile(testFile, []byte("firmware data"), 0644)
	assert.NoError(t, err)

	// Upsert using software name instead of ID
	opt := softwareversions.CreateVersionOptions{
		SoftwareName: softwareName,
		SoftwareType: softwareType,
		Version:      "2.1.0",
		File: softwareversions.UploadFileOptions{
			FilePath: testFile,
			Name:     "firmware.bin",
		},
	}

	result := client.Repository.Software.Versions.UpsertByVersion(ctx, opt)
	assert.NoError(t, result.Err)
	assert.Equal(t, "Created", string(result.Status))

	// Cleanup
	_ = client.Repository.Software.Versions.Delete(ctx, result.Data.ID(), softwareversions.DeleteOptions{})
}

func Test_SoftwareVersionUpsertByVersion_MultipleVersions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a software item
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "apt"

	softwareResult := client.Repository.Software.GetOrCreateByName(ctx, softwareName, softwareType, map[string]any{
		"name":         softwareName,
		"type":         "c8y_Software",
		"softwareType": softwareType,
	})
	assert.NoError(t, softwareResult.Err)
	softwareID := softwareResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Software.Delete(ctx, softwareID, softwareitems.DeleteOptions{})
	})

	// Create temporary files
	tempDir := t.TempDir()

	// Create version 1.0.0
	testFile1 := filepath.Join(tempDir, "v1.deb")
	err := os.WriteFile(testFile1, []byte("version 1.0.0"), 0644)
	assert.NoError(t, err)

	result1 := client.Repository.Software.Versions.UpsertByVersion(ctx, softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    "1.0.0",
		File: softwareversions.UploadFileOptions{
			FilePath: testFile1,
			Name:     "v1.deb",
		},
	})
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))

	// Create version 2.0.0
	testFile2 := filepath.Join(tempDir, "v2.deb")
	err = os.WriteFile(testFile2, []byte("version 2.0.0"), 0644)
	assert.NoError(t, err)

	result2 := client.Repository.Software.Versions.UpsertByVersion(ctx, softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    "2.0.0",
		File: softwareversions.UploadFileOptions{
			FilePath: testFile2,
			Name:     "v2.deb",
		},
	})
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Created", string(result2.Status))

	// Both versions should exist
	assert.NotEqual(t, result1.Data.ID(), result2.Data.ID(), "Different versions should have different IDs")

	// Update version 1.0.0
	testFile1Updated := filepath.Join(tempDir, "v1-updated.deb")
	err = os.WriteFile(testFile1Updated, []byte("version 1.0.0 updated"), 0644)
	assert.NoError(t, err)

	result3 := client.Repository.Software.Versions.UpsertByVersion(ctx, softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    "1.0.0",
		File: softwareversions.UploadFileOptions{
			FilePath: testFile1Updated,
			Name:     "v1-updated.deb",
		},
	})
	assert.NoError(t, result3.Err)
	assert.Equal(t, "Updated", string(result3.Status))
	assert.Equal(t, result1.Data.ID(), result3.Data.ID(), "Should update same version")

	// Cleanup
	_ = client.Repository.Software.Versions.Delete(ctx, result1.Data.ID(), softwareversions.DeleteOptions{})
	_ = client.Repository.Software.Versions.Delete(ctx, result2.Data.ID(), softwareversions.DeleteOptions{})
}
