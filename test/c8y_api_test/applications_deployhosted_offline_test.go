package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeWebApp creates a web application folder with an index.html and, when
// manifest is not empty, a cumulocity.json.
func writeWebApp(t *testing.T, dir string, manifest string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>hello</html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644))
	if manifest != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, applications.HostedManifestFile), []byte(manifest), 0o644))
	}
	return dir
}

// writeZip writes a zip file with the given entries.
func writeZip(t *testing.T, path string, entries map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, contents := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(contents))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	return path
}

func zipEntryNames(t *testing.T, data []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func newDeployHostedClient(t *testing.T) (*api.Client, *fakeserver.FakeServer) {
	t.Helper()
	testcore.SkipLive(t)
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	require.NotNil(t, srv)
	return client, srv
}

func countApplicationsByName(t *testing.T, client *api.Client, name string) int {
	t.Helper()
	res := client.Applications.List(context.Background(), applications.ListOptions{Name: name, Type: applications.TypeHosted})
	require.NoError(t, res.Err)
	items, err := res.Collect()
	require.NoError(t, err)
	return len(items)
}

func Test_Applications_DeployHosted_NewFromFolder(t *testing.T) {
	client, srv := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "webapp" + testingutils.RandomString(6)
	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), `{"name":"`+appName+`","key":"`+appName+`-key","contextPath":"`+appName+`-ctx"}`)

	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: dir})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)

	// Properties come from the manifest
	assert.Equal(t, appName, result.Data.Name())
	assert.Equal(t, appName+"-key", result.Data.Key())
	assert.Equal(t, applications.TypeHosted, result.Data.Type())
	assert.Equal(t, appName+"-ctx", result.Data.Get("contextPath").String())

	// The uploaded binary is activated
	binaryID, _ := result.Meta["binaryId"].(string)
	require.NotEmpty(t, binaryID)
	assert.Equal(t, binaryID, result.Data.Get("activeVersionId").String())
	assert.Equal(t, true, result.Meta["created"])
	assert.Equal(t, true, result.Meta["activated"])

	// The folder contents (not the folder itself) are zipped
	assert.Equal(t,
		[]string{"assets/", "assets/app.js", applications.HostedManifestFile, "index.html"},
		zipEntryNames(t, srv.BinaryData[binaryID]),
	)
}

func Test_Applications_DeployHosted_NewFromZip(t *testing.T) {
	client, srv := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "zipapp" + testingutils.RandomString(6)
	zipPath := writeZip(t, filepath.Join(t.TempDir(), appName+"-1.2.3.zip"), map[string]string{
		"index.html": "<html>zip</html>",
	})

	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: zipPath})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)

	// Name is taken from the file name without the version; key and context
	// path default to the name
	assert.Equal(t, appName, result.Data.Name())
	assert.Equal(t, appName+"-application-key", result.Data.Key())
	assert.Equal(t, appName, result.Data.Get("contextPath").String())

	// The zip is uploaded as is
	binaryID, _ := result.Meta["binaryId"].(string)
	expected, err := os.ReadFile(zipPath)
	require.NoError(t, err)
	assert.Equal(t, expected, srv.BinaryData[binaryID])
	assert.Equal(t, binaryID, result.Data.Get("activeVersionId").String())
}

func Test_Applications_DeployHosted_ManifestInZip(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "zipmanifest" + testingutils.RandomString(6)
	zipPath := writeZip(t, filepath.Join(t.TempDir(), "dist.zip"), map[string]string{
		"index.html":                    "<html>zip</html>",
		applications.HostedManifestFile: `{"name":"` + appName + `","key":"custom-key"}`,
	})

	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: zipPath})
	require.NoError(t, result.Err)
	assert.Equal(t, appName, result.Data.Name())
	assert.Equal(t, "custom-key", result.Data.Key())
}

func Test_Applications_DeployHosted_UpdateExisting(t *testing.T) {
	client, srv := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "existing" + testingutils.RandomString(6)
	existing := client.Applications.Create(ctx, map[string]any{
		"name":        appName,
		"key":         appName + "-original-key",
		"type":        applications.TypeHosted,
		"contextPath": appName,
	})
	require.NoError(t, existing.Err)

	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")
	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
		Path: dir,
		Name: appName,
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusUpdated, result.Status)
	assert.Equal(t, existing.Data.ID(), result.Data.ID())
	assert.Equal(t, false, result.Meta["created"])

	// Existing properties are kept, only the binary is replaced
	assert.Equal(t, appName+"-original-key", result.Data.Key())
	binaryID, _ := result.Meta["binaryId"].(string)
	require.NotEmpty(t, srv.BinaryData[binaryID])
	assert.Equal(t, binaryID, result.Data.Get("activeVersionId").String())
	assert.Equal(t, 1, countApplicationsByName(t, client, appName))
}

func Test_Applications_DeployHosted_IgnoresAppOfOtherOwner(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "owned" + testingutils.RandomString(6)
	parentApp := client.Applications.Create(ctx, map[string]any{
		"name":  appName,
		"key":   appName + "-key",
		"type":  applications.TypeHosted,
		"owner": map[string]any{"tenant": map[string]any{"id": "management"}},
	})
	require.NoError(t, parentApp.Err)

	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")
	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
		Path:  dir,
		Name:  appName,
		Owner: client.Auth.Tenant,
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)
	assert.NotEqual(t, parentApp.Data.ID(), result.Data.ID())
}

func Test_Applications_DeployHosted_SkipActivation(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "skipact" + testingutils.RandomString(6)
	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")
	first := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: dir, Name: appName})
	require.NoError(t, first.Err)
	activeID := first.Data.Get("activeVersionId").String()

	second := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
		Path:           dir,
		Name:           appName,
		SkipActivation: true,
	})
	require.NoError(t, second.Err)
	assert.Equal(t, op.StatusUpdated, second.Status)
	assert.Equal(t, false, second.Meta["activated"])
	assert.NotEqual(t, activeID, second.Meta["binaryId"])

	app := client.Applications.Get(ctx, first.Data.ID())
	require.NoError(t, app.Err)
	assert.Equal(t, activeID, app.Data.Get("activeVersionId").String())
}

func Test_Applications_DeployHosted_SkipUpload(t *testing.T) {
	client, srv := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "skipup" + testingutils.RandomString(6)
	before := srv.AppBinaries.Count()
	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Name: appName, SkipUpload: true})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)
	assert.Equal(t, appName, result.Data.Name())
	assert.Equal(t, before, srv.AppBinaries.Count())
}

func Test_Applications_DeployHosted_KeepBinaries(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "rotate" + testingutils.RandomString(6)
	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")

	uploaded := []string{}
	for range 4 {
		result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
			Path:         dir,
			Name:         appName,
			KeepBinaries: 2,
		})
		require.NoError(t, result.Err)
		uploaded = append(uploaded, result.Meta["binaryId"].(string))
	}

	binaries := client.Applications.ListBinaries(ctx, "name:"+appName)
	require.NoError(t, binaries.Err)
	remaining := []string{}
	for item, err := range binaries.Items() {
		require.NoError(t, err)
		remaining = append(remaining, item.ID())
	}
	// Only the two newest binaries are kept
	assert.ElementsMatch(t, uploaded[2:], remaining)
}

func Test_Applications_DeployHosted_KeepBinariesKeepsActive(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	appName := "rotateactive" + testingutils.RandomString(6)
	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")

	first := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: dir, Name: appName})
	require.NoError(t, first.Err)
	activeID := first.Meta["binaryId"].(string)

	var last string
	for range 2 {
		res := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
			Path:           dir,
			Name:           appName,
			SkipActivation: true,
			KeepBinaries:   1,
		})
		require.NoError(t, res.Err)
		last = res.Meta["binaryId"].(string)
	}

	binaries := client.Applications.ListBinaries(ctx, first.Data.ID())
	require.NoError(t, binaries.Err)
	remaining := []string{}
	for item, err := range binaries.Items() {
		require.NoError(t, err)
		remaining = append(remaining, item.ID())
	}
	// The active binary and the new upload are never deleted
	assert.ElementsMatch(t, []string{activeID, last}, remaining)
}

func Test_Applications_DeployHosted_InvalidPath(t *testing.T) {
	client, _ := newDeployHostedClient(t)
	ctx := context.Background()

	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{
		Path: filepath.Join(t.TempDir(), "does-not-exist"),
	})
	require.Error(t, result.Err)

	result = client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Name: "noPath"})
	require.Error(t, result.Err)
}

func Test_Applications_DeployHosted_DryRun(t *testing.T) {
	client, srv := newDeployHostedClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	appName := "dryrun" + testingutils.RandomString(6)
	dir := writeWebApp(t, filepath.Join(t.TempDir(), "build"), "")
	appsBefore := srv.Applications.Count()
	binariesBefore := srv.AppBinaries.Count()

	result := client.Applications.DeployHosted(ctx, applications.DeployHostedOptions{Path: dir, Name: appName})
	require.NoError(t, result.Err)
	assert.Equal(t, true, result.Meta["created"])
	assert.Nil(t, result.Meta["binaryId"])

	// Nothing is created or uploaded
	assert.Equal(t, appsBefore, srv.Applications.Count())
	assert.Equal(t, binariesBefore, srv.AppBinaries.Count())
}
