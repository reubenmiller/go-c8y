package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_SoftwareUpsertByName_Create(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := testingutils.RandomString(16)
	softwareType := "application"

	body := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  "Initial description",
	}

	// First upsert should create
	result1 := client.Repository.Software.UpsertByName(ctx, name, softwareType, body)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.False(t, result1.Meta["found"].(bool))

	id1 := result1.Data.ID()

	// Second upsert with same name/type should update
	body["description"] = "Updated description"
	result2 := client.Repository.Software.UpsertByName(ctx, name, softwareType, body)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.True(t, result2.Meta["found"].(bool))

	id2 := result2.Data.ID()
	assert.Equal(t, id1, id2, "Should update same software item")

	// Verify description was updated
	fetchResult := client.Repository.Software.Get(ctx, id2, softwareitems.GetOptions{})
	assert.NoError(t, fetchResult.Err)
	assert.Equal(t, "Updated description", fetchResult.Data.Description)

	// Cleanup
	_ = client.Repository.Software.Delete(ctx, id1, softwareitems.DeleteOptions{})
}

func Test_SoftwareUpsertWith_ArchitectureSupport(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := testingutils.RandomString(16)
	softwareType := "apt"
	arch := "arm64"

	body := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  "ARM64 package",
		"deviceType":   arch,
	}

	query := "name eq '" + name + "' and softwareType eq '" + softwareType + "' and deviceType eq '" + arch + "'"

	// First upsert should create
	result1 := client.Repository.Software.UpsertWith(ctx, query, body)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))

	id1 := result1.Data.ID()

	// Second upsert should update
	body["description"] = "Updated ARM64 package"
	result2 := client.Repository.Software.UpsertWith(ctx, query, body)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.Equal(t, id1, result2.Data.ID(), "Should update same software item")

	// Verify description and deviceType
	fetchResult := client.Repository.Software.Get(ctx, id1, softwareitems.GetOptions{})
	assert.NoError(t, fetchResult.Err)
	assert.Equal(t, "Updated ARM64 package", fetchResult.Data.Description)
	assert.Equal(t, arch, fetchResult.Data.Get("deviceType").String())

	// Create another software with different architecture
	arch2 := "amd64"
	body2 := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  "AMD64 package",
		"deviceType":   arch2,
	}
	query2 := "name eq '" + name + "' and softwareType eq '" + softwareType + "' and deviceType eq '" + arch2 + "'"

	result3 := client.Repository.Software.UpsertWith(ctx, query2, body2)
	assert.NoError(t, result3.Err)
	assert.Equal(t, "Created", string(result3.Status))
	assert.NotEqual(t, id1, result3.Data.ID(), "Should create different software item for different architecture")

	// Cleanup
	_ = client.Repository.Software.Delete(ctx, id1, softwareitems.DeleteOptions{})
	_ = client.Repository.Software.Delete(ctx, result3.Data.ID(), softwareitems.DeleteOptions{})
}

func Test_SoftwareUpsertByName_MetadataUpdate(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := testingutils.RandomString(16)
	softwareType := "firmware"

	// Create with initial metadata
	body1 := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  "Version 1.0",
		"custom_field": "value1",
	}

	result1 := client.Repository.Software.UpsertByName(ctx, name, softwareType, body1)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))

	id := result1.Data.ID()

	// Update with new metadata
	body2 := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  "Version 2.0",
		"custom_field": "value2",
		"new_field":    "new_value",
	}

	result2 := client.Repository.Software.UpsertByName(ctx, name, softwareType, body2)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.Equal(t, id, result2.Data.ID())

	// Verify metadata was updated
	fetchResult := client.Repository.Software.Get(ctx, id, softwareitems.GetOptions{})
	assert.NoError(t, fetchResult.Err)
	assert.Equal(t, "Version 2.0", fetchResult.Data.Description)

	// Cleanup
	_ = client.Repository.Software.Delete(ctx, id, softwareitems.DeleteOptions{})
}
