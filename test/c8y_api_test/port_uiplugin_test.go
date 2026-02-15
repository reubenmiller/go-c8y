package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/plugins/versions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uiPluginURLApp1Version1 = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.4.3/cloud-http-proxy-ui.zip"
var uiPluginURLApp1Version2 = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.5.0/cloud-http-proxy-ui.zip"

// downloadFile downloads a file from a URL and saves it to the provided WriteCloser
func downloadFile(u string, out io.WriteCloser) error {
	defer out.Close()
	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("failed to download file from url. %w", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// createTempFile creates a temporary file which will be cleaned up at the end of the test
func createTempFile(t *testing.T, name string) *os.File {
	file, err := os.CreateTemp("", "*_"+name)
	require.NoError(t, err)
	t.Cleanup(func() {
		file.Close()
		os.Remove(file.Name())
	})
	return file
}

func Test_UIPlugin_CreateWithVersions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appName := testingutils.RandomString(12)

	//
	// Download Version 1
	file1 := createTempFile(t, "examplePlugin1.zip")
	err := downloadFile(uiPluginURLApp1Version1, file1)
	require.NoError(t, err)

	// Read plugin manifest from zip file
	plugin, err := client.UIPlugins.NewPluginFromFile(file1.Name())
	require.NoError(t, err)
	assert.NotEmpty(t, plugin.Name, "Plugin name should not be empty")
	assert.NotEmpty(t, plugin.Key, "Plugin key should not be empty")

	// Use unique name to avoid name clashes
	plugin.Name = appName
	plugin.Key = appName + "-key"
	plugin.ContextPath = appName

	// Create the plugin application
	createResult := client.UIPlugins.Create(ctx, plugin)
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	pluginID := createResult.Data.ID()
	assert.NotEmpty(t, pluginID, "Plugin ID should not be empty")

	t.Cleanup(func() {
		client.UIPlugins.Delete(context.Background(), pluginID)
	})

	// Upload first version (2.4.3) with tag1
	version1Result := client.UIPluginVersions.Create(ctx, pluginID, versions.CreateOptions{
		Version:  "2.4.3",
		Tags:     []string{"tag1"},
		Filename: file1.Name(),
	})
	require.NoError(t, version1Result.Err)
	tags := version1Result.Data.Tags()
	assert.Equal(t, 201, version1Result.HTTPStatus)
	assert.Equal(t, "2.4.3", version1Result.Data.Version())
	assert.NotEmpty(t, version1Result.Data.BinaryID())
	assert.Len(t, tags, 2, "Tags should contain 2 items")
	assert.Contains(t, tags, "tag1")
	assert.Contains(t, tags, "latest")

	//
	// Download Version 2
	file2 := createTempFile(t, "examplePlugin2.zip")
	err = downloadFile(uiPluginURLApp1Version2, file2)
	require.NoError(t, err)

	// Upload second version (2.5.0) with tag2 and latest
	version2Result := client.UIPluginVersions.Create(ctx, pluginID, versions.CreateOptions{
		Version:  "2.5.0",
		Tags:     []string{"latest", "tag2"},
		Filename: file2.Name(),
	})
	require.NoError(t, version2Result.Err)
	assert.Equal(t, 201, version2Result.HTTPStatus)
	assert.Equal(t, "2.5.0", version2Result.Data.Version())
	assert.Len(t, version2Result.Data.Tags(), 2, "Tags should contain 2 items")
	assert.Contains(t, version2Result.Data.Tags(), "tag2")
	assert.Contains(t, version2Result.Data.Tags(), "latest")
}
