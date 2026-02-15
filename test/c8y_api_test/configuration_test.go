package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/configuration"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Configuration_Create(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	ctx := context.Background()

	name := "config-" + testingutils.RandomString(16)
	opt := configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "agentConfig",
		Description:       "Test configuration",
		DeviceType:        "thin-edge.io",
	}

	result := client.Repository.Configuration.Create(ctx, opt)
	assert.NoError(t, result.Err)
	assert.Equal(t, "Created", string(result.Status))
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, name, result.Data.Name())
	assert.Equal(t, "agentConfig", result.Data.ConfigurationType())
	assert.Equal(t, "thin-edge.io", result.Data.DeviceType())
	assert.Equal(t, "Test configuration", result.Data.Description())

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, result.Data.ID(), configuration.DeleteOptions{})
	})
}

func Test_Configuration_CreateWithFile(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-config.toml")
	err := os.WriteFile(testFile, []byte("# Test config content"), 0644)
	assert.NoError(t, err)

	name := "config-" + testingutils.RandomString(16)
	opt := configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "toml",
		Description:       "Configuration with file",
		DeviceType:        "linux",
		File: configuration.UploadFileOptions{
			FilePath: testFile,
			Name:     "test-config.toml",
		},
	}

	result := client.Repository.Configuration.Create(ctx, opt)
	assert.NoError(t, result.Err)
	assert.Equal(t, "Created", string(result.Status))
	assert.NotEmpty(t, result.Data.URL())

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, result.Data.ID(), configuration.DeleteOptions{})
	})
}

func Test_Configuration_CreateWithAdditionalProperties(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "config-" + testingutils.RandomString(16)
	opt := configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "custom",
		Description:       "Configuration with additional properties",
		AdditionalProperties: map[string]any{
			"c8y_CustomFragment": map[string]any{
				"customField1": "value1",
				"customField2": 42,
			},
			"owner":    "admin",
			"category": "testing",
		},
	}

	result := client.Repository.Configuration.Create(ctx, opt)
	assert.NoError(t, result.Err)
	assert.Equal(t, "Created", string(result.Status))
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, name, result.Data.Name())

	// Verify additional properties are present
	customFragment := result.Data.Get("c8y_CustomFragment")
	assert.True(t, customFragment.Exists())
	assert.Equal(t, "value1", customFragment.Get("customField1").String())
	assert.Equal(t, int64(42), customFragment.Get("customField2").Int())

	assert.Equal(t, "admin", result.Data.Get("owner").String())
	assert.Equal(t, "testing", result.Data.Get("category").String())

	// Verify standard fields cannot be overridden
	assert.Equal(t, "c8y_ConfigurationDump", result.Data.Type())

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, result.Data.ID(), configuration.DeleteOptions{})
	})
}

func Test_Configuration_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a configuration first
	name := "config-" + testingutils.RandomString(16)
	createResult := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "json",
		Description:       "Get test",
	})
	assert.NoError(t, createResult.Err)
	configID := createResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Get by ID
	result := client.Repository.Configuration.Get(ctx, configID, configuration.GetOptions{})
	assert.NoError(t, result.Err)
	assert.Equal(t, configID, result.Data.ID())
	assert.Equal(t, name, result.Data.Name())
	assert.Equal(t, "json", result.Data.ConfigurationType())
}

func Test_Configuration_Update(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a configuration
	name := "config-" + testingutils.RandomString(16)
	createResult := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "yaml",
		Description:       "Original description",
	})
	assert.NoError(t, createResult.Err)
	configID := createResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Update
	updateResult := client.Repository.Configuration.Update(ctx, configID, map[string]any{
		"description": "Updated description",
	})
	assert.NoError(t, updateResult.Err)
	assert.Equal(t, configID, updateResult.Data.ID())
	assert.Equal(t, "Updated description", updateResult.Data.Description())
}

func Test_Configuration_Delete(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a configuration
	name := "config-" + testingutils.RandomString(16)
	createResult := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "ini",
	})
	assert.NoError(t, createResult.Err)
	configID := createResult.Data.ID()

	// Delete
	deleteResult := client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{
		ForceCascade: true,
	})
	assert.NoError(t, deleteResult.Err)

	// Verify deletion
	getResult := client.Repository.Configuration.Get(ctx, configID, configuration.GetOptions{})
	assert.Error(t, getResult.Err)
	assert.Equal(t, 404, getResult.HTTPStatus)
}

func Test_Configuration_List(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a configuration
	name := "list-test-" + testingutils.RandomString(16)
	result1 := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "json",
		DeviceType:        "linux",
	})
	assert.NoError(t, result1.Err)

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, result1.Data.ID(), configuration.DeleteOptions{})
	})

	// List by name
	listResult := client.Repository.Configuration.List(ctx, configuration.ListOptions{
		Name: name,
	})
	assert.NoError(t, listResult.Err)

	found := false
	for doc := range listResult.Data.Iter() {
		item := jsonmodels.NewConfiguration(doc.Bytes())
		if item.ID() == result1.Data.ID() {
			found = true
			assert.Equal(t, name, item.Name())
			break
		}
	}
	assert.True(t, found, "Should find created configuration")
}

func Test_Configuration_GetOrCreate(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "getorcreate-" + testingutils.RandomString(16)
	configurationType := "json"

	// First call should create
	result1 := client.Repository.Configuration.GetOrCreate(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
		Description:       "GetOrCreate test",
	})
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	configID := result1.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Second call should get existing
	result2 := client.Repository.Configuration.GetOrCreate(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
		Description:       "GetOrCreate test",
	})
	assert.NoError(t, result2.Err)
	assert.Equal(t, "OK", string(result2.Status))
	assert.Equal(t, configID, result2.Data.ID())
}

func Test_Configuration_UpsertByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "upsert-" + testingutils.RandomString(16)
	configurationType := "yaml"

	// First upsert should create
	result1 := client.Repository.Configuration.UpsertByName(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
		Description:       "Initial description",
		DeviceType:        "linux",
	})
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.False(t, result1.Meta["found"].(bool))
	configID := result1.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Second upsert should update
	result2 := client.Repository.Configuration.UpsertByName(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
		Description:       "Updated description",
		DeviceType:        "windows",
	})
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.True(t, result2.Meta["found"].(bool))
	assert.Equal(t, configID, result2.Data.ID())
	assert.Equal(t, "Updated description", result2.Data.Description())
	assert.Equal(t, "windows", result2.Data.DeviceType())
}

func Test_Configuration_UpsertByName_WithFile(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create temporary files
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "config-v1.toml")
	err := os.WriteFile(testFile1, []byte("# Config v1"), 0644)
	assert.NoError(t, err)

	name := "upsert-file-" + testingutils.RandomString(16)

	// First upsert with file
	result1 := client.Repository.Configuration.UpsertByName(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "toml",
		File: configuration.UploadFileOptions{
			FilePath: testFile1,
			Name:     "config-v1.toml",
		},
	})
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.NotEmpty(t, result1.Data.URL())
	configID := result1.Data.ID()
	originalURL := result1.Data.URL()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Second upsert with different file
	testFile2 := filepath.Join(tempDir, "config-v2.toml")
	err = os.WriteFile(testFile2, []byte("# Config v2 - updated"), 0644)
	assert.NoError(t, err)

	result2 := client.Repository.Configuration.UpsertByName(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "toml",
		File: configuration.UploadFileOptions{
			FilePath: testFile2,
			Name:     "config-v2.toml",
		},
	})
	assert.NoError(t, result2.Err)
	assert.Equal(t, "Updated", string(result2.Status))
	assert.Equal(t, configID, result2.Data.ID())
	assert.NotEmpty(t, result2.Data.URL())
	assert.NotEqual(t, originalURL, result2.Data.URL(), "URL should change after uploading new file")
}

func Test_Configuration_Resolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "resolver-name-" + testingutils.RandomString(16)
	configurationType := "json"

	createResult := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
	})
	assert.NoError(t, createResult.Err)
	configID := createResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Resolve by name
	identifier := configuration.NewRef().ByName(name)
	getResult := client.Repository.Configuration.Get(ctx, identifier, configuration.GetOptions{})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, configID, getResult.Data.ID())
	assert.Equal(t, name, getResult.Data.Name())
}

func Test_Configuration_Resolver_ByNameAndType(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "resolver-nametype-" + testingutils.RandomString(16)
	configurationType := "yaml"

	// Create two configs with same name but different types
	result1 := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: configurationType,
		Description:       "Type yaml",
	})
	assert.NoError(t, result1.Err)

	result2 := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "json",
		Description:       "Type json",
	})
	assert.NoError(t, result2.Err)

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, result1.Data.ID(), configuration.DeleteOptions{})
		client.Repository.Configuration.Delete(ctx, result2.Data.ID(), configuration.DeleteOptions{})
	})

	// Resolve by name and type
	identifier := configuration.NewRef().ByName(name, configurationType)
	getResult := client.Repository.Configuration.Get(ctx, identifier, configuration.GetOptions{})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, result1.Data.ID(), getResult.Data.ID())
	assert.Equal(t, configurationType, getResult.Data.ConfigurationType())
}

func Test_Configuration_Resolver_ByQuery(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceType := "resolver-query-" + testingutils.RandomString(8)
	name := "config-" + testingutils.RandomString(16)

	createResult := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
		Name:              name,
		ConfigurationType: "ini",
		DeviceType:        deviceType,
	})
	assert.NoError(t, createResult.Err)
	configID := createResult.Data.ID()

	t.Cleanup(func() {
		client.Repository.Configuration.Delete(ctx, configID, configuration.DeleteOptions{})
	})

	// Resolve by query
	identifier := configuration.NewRef().ByQuery("deviceType eq '" + deviceType + "'")
	getResult := client.Repository.Configuration.Get(ctx, identifier, configuration.GetOptions{})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, configID, getResult.Data.ID())
	assert.Equal(t, deviceType, getResult.Data.DeviceType())
}
