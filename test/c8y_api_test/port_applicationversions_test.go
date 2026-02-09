package c8y_api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	appversions "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/applications/versions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/ui/plugins"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uiExamplePluginURL = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.5.0/cloud-http-proxy-ui.zip"

// createTestPlugin creates a test UI plugin with a version for testing
func createTestPlugin(t *testing.T, client *c8y_api.Client, version string, tags []string) string {
	ctx := context.Background()
	appName := testingutils.RandomString(12)

	// Create the plugin application
	plugin := plugins.NewPlugin(appName)
	plugin.Key = appName + "-key"
	plugin.ContextPath = appName

	createResult := client.UIPlugins.Create(ctx, plugin)
	require.NoError(t, createResult.Err)
	pluginID := createResult.Data.ID()

	// Upload version
	versionResult := client.ApplicationVersions.CreateFromFile(ctx, pluginID, uiExamplePluginURL, version, tags)
	require.NoError(t, versionResult.Err)

	t.Cleanup(func() {
		client.UIPlugins.Delete(context.Background(), pluginID)
	})

	return pluginID
}

func Test_ApplicationVersions_GetVersions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appID := createTestPlugin(t, client, "1.0.0", []string{"latest"})

	// List versions
	result := client.ApplicationVersions.List(ctx, appID, appversions.ListOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	versions, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.NotEmpty(t, versions, "At least one application version should be found")
	assert.Equal(t, "1.0.0", versions[0].Version())
	assert.Len(t, versions[0].Tags(), 1, "Tags should be present")
	assert.Contains(t, versions[0].Tags(), "latest")
}

func Test_ApplicationVersions_GetVersionByTag(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appID := createTestPlugin(t, client, "1.0.1", []string{"latest", "tag1"})

	// List versions by tag
	result := client.ApplicationVersions.ListByTag(ctx, appID, "tag1")

	require.NoError(t, result.Err)

	versions, err := op.ToSliceR(result)
	require.NoError(t, err)

	assert.Equal(t, 200, result.HTTPStatus)
	require.Len(t, versions, 1)
	// TODO: Should the a collection of results also allow the user to get the first item but just doing result.Data.Version()?
	assert.Equal(t, "1.0.1", versions[0].Version())
	assert.Len(t, versions[0].Tags(), 2)
	assert.Contains(t, versions[0].Tags(), "latest")
	assert.Contains(t, versions[0].Tags(), "tag1")
}

func Test_ApplicationVersions_GetVersionByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appID := createTestPlugin(t, client, "1.0.2", []string{"tag1"})

	// Get version by name
	result := client.ApplicationVersions.ListByVersion(ctx, appID, "1.0.2")

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	versions, err := op.ToSliceR(result)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "1.0.2", versions[0].Version())
	assert.Len(t, versions[0].Tags(), 2, "Should have 2 tags")
	assert.Contains(t, versions[0].Tags(), "latest", "Latest is automatically added when activated")
	assert.Contains(t, versions[0].Tags(), "tag1")
}

func Test_ApplicationVersions_CRUD_Extension(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appName := testingutils.RandomString(12)

	// Create the plugin application
	plugin := plugins.NewPlugin(appName)
	plugin.Key = appName + "-key"
	plugin.ContextPath = appName

	createResult := client.UIPlugins.Create(ctx, plugin)
	require.NoError(t, createResult.Err)
	pluginID := createResult.Data.ID()

	t.Cleanup(func() {
		client.UIPlugins.Delete(context.Background(), pluginID)
	})

	// Create first version
	version1Result := client.ApplicationVersions.CreateFromFile(ctx, pluginID, uiExamplePluginURL, "2.5.0", []string{"latest", "tag1"})
	require.NoError(t, version1Result.Err)
	assert.Equal(t, 201, version1Result.HTTPStatus)
	assert.Equal(t, "2.5.0", version1Result.Data.Version())
	assert.Len(t, version1Result.Data.Tags(), 2)
	assert.Contains(t, version1Result.Data.Tags(), "latest")
	assert.Contains(t, version1Result.Data.Tags(), "tag1")

	// Create second version
	version2Result := client.ApplicationVersions.CreateFromFile(ctx, pluginID, uiExamplePluginURL, "2.5.1", []string{"latest", "tagA"})
	require.NoError(t, version2Result.Err)
	assert.Equal(t, 201, version2Result.HTTPStatus)
	assert.Equal(t, "2.5.1", version2Result.Data.Version())
	assert.Len(t, version2Result.Data.Tags(), 2)
	assert.Contains(t, version2Result.Data.Tags(), "latest")
	// Tags are lowercased by the platform
	containsTagA := false
	for _, tag := range version2Result.Data.Tags() {
		if strings.EqualFold(tag, "taga") {
			containsTagA = true
			break
		}
	}
	assert.True(t, containsTagA, "Should contain tag 'taga' or 'tagA'")

	// Update tags on first version
	updatedResult := client.ApplicationVersions.Update(ctx, pluginID, "2.5.0", []string{"tag2", "tag3", "latest"})
	require.NoError(t, updatedResult.Err)
	assert.Equal(t, 200, updatedResult.HTTPStatus)
	assert.Len(t, updatedResult.Data.Tags(), 3)
	assert.Contains(t, updatedResult.Data.Tags(), "tag2")
	assert.Contains(t, updatedResult.Data.Tags(), "tag3")
	assert.Contains(t, updatedResult.Data.Tags(), "latest")

	// Activate version (set active version to empty to deactivate)
	activateResult := client.UIPlugins.Activate(ctx, pluginID, updatedResult.Data.BinaryID())
	require.NoError(t, activateResult.Err)
	assert.Equal(t, 200, activateResult.HTTPStatus)

	// Delete by version (the non-active version)
	_, err := client.ApplicationVersions.DeleteByVersion(ctx, pluginID, "2.5.1")
	require.NoError(t, err)
}
