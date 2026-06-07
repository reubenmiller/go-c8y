package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	appversions "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications/versions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/ui/plugins"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPluginAndApp(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	zipBuf := bytes.NewBuffer(nil)
	// Minimal zip-shaped header so that uploading works.  The fake server does
	// not actually unzip the file.
	zipBuf.Write([]byte("PK\x03\x04dummy-content"))
	return "", zipBuf
}

func Test_ApplicationVersions_FullLifecycle(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "vplug" + testingutils.RandomString(6)
	p := plugins.NewPlugin(name)
	pluginRes := client.UIPlugins.Create(ctx, p)
	require.NoError(t, pluginRes.Err)
	pluginID := pluginRes.Data.ID()
	t.Cleanup(func() {
		client.UIPlugins.Delete(ctx, pluginID)
	})

	// Use Create() directly with an in-memory reader to avoid needing a file
	// or a network download.
	_, zipBuf := newPluginAndApp(t)
	createRes := client.ApplicationVersions.Create(ctx, pluginID, appversions.CreateOptions{
		Version: "1.0.0",
		Tags:    []string{"latest", "stable"},
		File: appversions.UploadFileOptions{
			FilePath: "app.zip",
			Reader:   bytes.NewReader(zipBuf.Bytes()),
		},
	})
	require.NoError(t, createRes.Err)

	listRes := client.ApplicationVersions.List(ctx, pluginID, appversions.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.ApplicationVersions.ListAll(ctx, pluginID, appversions.ListOptions{})
	require.NoError(t, itAll.Err())

	byVersion := client.ApplicationVersions.ListByVersion(ctx, pluginID, "1.0.0")
	require.NoError(t, byVersion.Err)

	byTag := client.ApplicationVersions.ListByTag(ctx, pluginID, "latest")
	require.NoError(t, byTag.Err)

	upd := client.ApplicationVersions.Update(ctx, pluginID, "1.0.0", []string{"latest"})
	require.NoError(t, upd.Err)

	// Create a second version so we can exercise DeleteByTag while a tag
	// still exists.
	createRes2 := client.ApplicationVersions.Create(ctx, pluginID, appversions.CreateOptions{
		Version: "2.0.0",
		Tags:    []string{"beta"},
		File: appversions.UploadFileOptions{
			FilePath: "app.zip",
			Reader:   bytes.NewReader(zipBuf.Bytes()),
		},
	})
	require.NoError(t, createRes2.Err)

	del := client.ApplicationVersions.DeleteByVersion(ctx, pluginID, "1.0.0")
	require.NoError(t, del.Err)

	delTag := client.ApplicationVersions.DeleteByTag(ctx, pluginID, "beta")
	require.NoError(t, delTag.Err)
}

func Test_ApplicationVersions_CreateFromFile_LocalPath(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "vplug2" + testingutils.RandomString(6)
	pluginRes := client.UIPlugins.Create(ctx, plugins.NewPlugin(name))
	require.NoError(t, pluginRes.Err)
	pluginID := pluginRes.Data.ID()
	t.Cleanup(func() {
		client.UIPlugins.Delete(ctx, pluginID)
	})

	// Write a small dummy file to disk
	dir := t.TempDir()
	fpath := filepath.Join(dir, "plugin.zip")
	require.NoError(t, os.WriteFile(fpath, []byte("PK\x03\x04dummy"), 0o600))

	res := client.ApplicationVersions.CreateFromFile(ctx, pluginID, fpath, "1.2.3", []string{"latest"})
	require.NoError(t, res.Err)
}

func Test_ApplicationVersions_CreateFromFile_URL(t *testing.T) {
	// Spin up a local server that serves a dummy "binary".
	binary := []byte("PK\x03\x04remote-content")
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(binary)
	}))
	defer httpSrv.Close()

	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "vplug3" + testingutils.RandomString(6)
	pluginRes := client.UIPlugins.Create(ctx, plugins.NewPlugin(name))
	require.NoError(t, pluginRes.Err)
	pluginID := pluginRes.Data.ID()
	t.Cleanup(func() {
		client.UIPlugins.Delete(ctx, pluginID)
	})

	res := client.ApplicationVersions.CreateFromFile(ctx, pluginID, httpSrv.URL+"/plugin.zip", "2.0.0", []string{"latest"})
	require.NoError(t, res.Err)
}

func Test_ApplicationVersions_CreateFromFile_MissingFile(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "vplug4" + testingutils.RandomString(6)
	pluginRes := client.UIPlugins.Create(ctx, plugins.NewPlugin(name))
	require.NoError(t, pluginRes.Err)
	pluginID := pluginRes.Data.ID()
	t.Cleanup(func() {
		client.UIPlugins.Delete(ctx, pluginID)
	})

	res := client.ApplicationVersions.CreateFromFile(ctx, pluginID, "/nonexistent/file/path.zip", "0.0.0", nil)
	assert.Error(t, res.Err)
}

func Test_Applications_Upload(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "upapp" + testingutils.RandomString(6)
	createRes := client.Applications.Create(ctx, map[string]any{
		"name":        name,
		"key":         name + "-key",
		"type":        "HOSTED",
		"contextPath": name,
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Applications.Delete(ctx, id, applications.DeleteOptions{})
	})

	uploadRes := client.Applications.Upload(ctx, id, applications.UploadFileOptions{
		Name:   "binary.zip",
		Reader: bytes.NewReader([]byte("PK\x03\x04dummy")),
	})
	_ = uploadRes
}
