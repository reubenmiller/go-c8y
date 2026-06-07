package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/ui/applicationplugins"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/ui/plugins"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UIPlugins_NewPlugin(t *testing.T) {
	p := plugins.NewPlugin("myplugin")
	assert.Equal(t, "myplugin", p.Name)
	assert.Equal(t, "myplugin-key", p.Key)
	assert.Equal(t, "myplugin", p.ContextPath)
	assert.Equal(t, plugins.ApplicationTypeHosted, p.Type)
	require.NotNil(t, p.Manifest)
	assert.True(t, *p.Manifest.IsPackage)
	assert.Equal(t, "plugin", p.Manifest.Package)
}

func Test_UIPlugins_ManifestHelpers(t *testing.T) {
	m := &plugins.Manifest{}
	m.WithIsPackage(false).WithPackage("pkg")
	assert.False(t, *m.IsPackage)
	assert.Equal(t, "pkg", m.Package)
}

func Test_UIPlugins_HasTag(t *testing.T) {
	assert.True(t, plugins.HasTag([]string{"latest", "tag1"}, "TAG1"))
	assert.False(t, plugins.HasTag([]string{"latest"}, "missing"))
}

func Test_UIPlugins_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "plug" + testingutils.RandomString(6)
	p := plugins.NewPlugin(name)
	createResult := client.UIPlugins.Create(ctx, p)
	require.NoError(t, createResult.Err)

	id := createResult.Data.ID()
	getResult := client.UIPlugins.Get(ctx, id)
	require.NoError(t, getResult.Err)
	assert.Equal(t, id, getResult.Data.ID())

	listRes := client.UIPlugins.List(ctx, plugins.ListOptions{})
	require.NoError(t, listRes.Err)
	_, err := op.ToSliceR(listRes)
	require.NoError(t, err)

	itAll := client.UIPlugins.ListAll(ctx, plugins.ListOptions{})
	require.NoError(t, itAll.Err())

	updRes := client.UIPlugins.Update(ctx, id, &plugins.Plugin{ContextPath: name + "-v2"})
	require.NoError(t, updRes.Err)

	actRes := client.UIPlugins.Activate(ctx, id, "binary-1")
	require.NoError(t, actRes.Err)

	delRes := client.UIPlugins.Delete(ctx, id)
	require.NoError(t, delRes.Err)
}

func Test_UIApplicationPlugins_Lifecycle(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a host application
	name := "host" + testingutils.RandomString(6)
	create := client.Applications.Create(ctx, map[string]any{
		"name":        name,
		"key":         name + "-key",
		"type":        "HOSTED",
		"contextPath": name,
	})
	require.NoError(t, create.Err)
	appID := create.Data.ID()
	t.Cleanup(func() {
		client.Applications.Delete(ctx, appID, applications.DeleteOptions{})
	})

	listRes := client.UIApplicationPlugins.List(ctx, appID, applicationplugins.ListOptions{})
	require.NoError(t, listRes.Err)

	installRes := client.UIApplicationPlugins.Install(ctx, appID, "plug-1")
	require.NoError(t, installRes.Err)

	updateRes := client.UIApplicationPlugins.Update(ctx, appID, []applicationplugins.PluginReference{
		{ID: "plug-1", Name: "p1", Version: "1.0"},
		{ID: "plug-2", Name: "p2", Version: "1.1"},
	})
	require.NoError(t, updateRes.Err)

	replaceRes := client.UIApplicationPlugins.Replace(ctx, appID, []applicationplugins.PluginReference{
		{ID: "plug-2"},
	})
	require.NoError(t, replaceRes.Err)

	delRes := client.UIApplicationPlugins.Delete(ctx, appID, "plug-2")
	require.NoError(t, delRes.Err)
}

func Test_UIApplicationPlugins_NewWrappers(t *testing.T) {
	ref := applicationplugins.NewPluginReference([]byte(`{"id":"p1","name":"plug","version":"1.0.0"}`))
	assert.Equal(t, "p1", ref.ID())
	assert.Equal(t, "plug", ref.Name())
	assert.Equal(t, "1.0.0", ref.Version())

	app := applicationplugins.NewApplicationWithPlugins([]byte(`{"id":"99","applicationBuilder":{"plugins":[{"id":"p1"}]}}`))
	assert.Equal(t, "99", app.ID())
	pl := app.Plugins()
	require.Len(t, pl, 1)
	assert.Equal(t, "p1", pl[0]["id"])
}
