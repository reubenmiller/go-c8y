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
	testcore.SkipLive(t)
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
	testcore.SkipLive(t)
	m := &plugins.Manifest{}
	m.WithIsPackage(false).WithPackage("pkg")
	assert.False(t, *m.IsPackage)
	assert.Equal(t, "pkg", m.Package)
}

func Test_UIPlugins_HasTag(t *testing.T) {
	testcore.SkipLive(t)
	assert.True(t, plugins.HasTag([]string{"latest", "tag1"}, "TAG1"))
	assert.False(t, plugins.HasTag([]string{"latest"}, "missing"))
}

func Test_UIPlugins_CRUD(t *testing.T) {
	testcore.SkipLive(t)
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
	_, err := listRes.Collect()
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
	testcore.SkipLive(t)
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	suffix := testingutils.RandomString(6)

	create := func(body map[string]any) string {
		res := client.Applications.Create(ctx, body)
		require.NoError(t, res.Err)
		id := res.Data.ID()
		t.Cleanup(func() { client.Applications.Delete(ctx, id, applications.DeleteOptions{}) })
		return id
	}
	// A host application with a remote that is not a plugin and another config field
	hostName := "host" + suffix
	appID := create(map[string]any{
		"name": hostName, "key": hostName + "-key", "type": "HOSTED", "contextPath": hostName,
		"config": map[string]any{
			"remotes": map[string]any{"legacy": []string{"LegacyModule"}},
			"other":   "kept",
		},
	})
	// A plugin with two versions
	plugin := "plug" + suffix
	create(map[string]any{
		"name": plugin, "key": plugin + "-key", "type": "HOSTED", "contextPath": plugin + "-ctx",
		"manifest": map[string]any{"isPackage": true, "remotes": map[string]any{plugin + "-ctx": []string{"PlugModule"}}},
		"applicationVersions": []map[string]any{
			{"version": "1.0.0", "tags": []string{}},
			{"version": "2.0.0", "tags": []string{"latest"}},
		},
	})
	remotes := func(res op.Result[applicationplugins.ApplicationWithPlugins]) map[string]any {
		t.Helper()
		require.NoError(t, res.Err)
		got := client.UIApplicationPlugins.List(ctx, appID)
		require.NoError(t, got.Err)
		assert.Equal(t, "kept", got.Data.Get("config.other").String(), "other config fields are kept")
		return got.Data.Get("config.remotes").Value().(map[string]any)
	}
	key := plugin + "-ctx@"

	// install a version (by name)
	r := remotes(client.UIApplicationPlugins.Install(ctx, appID, plugin+"@1.0.0"))
	assert.Equal(t, []any{"PlugModule"}, r[key+"1.0.0"])
	assert.Equal(t, []any{"LegacyModule"}, r["legacy"], "remotes that are not plugins are kept")

	// update all to the latest version
	r = remotes(client.UIApplicationPlugins.Update(ctx, appID, applicationplugins.UpdateOptions{UpdateAll: true}))
	assert.Contains(t, r, key+"2.0.0")
	assert.NotContains(t, r, key+"1.0.0")

	// list with the plugin details
	docs := 0
	for doc, err := range client.UIApplicationPlugins.ListPlugins(ctx, appID).Documents() {
		require.NoError(t, err)
		assert.Equal(t, plugin, doc.Get("name").String())
		assert.Equal(t, "2.0.0", doc.Get("version").String())
		docs++
	}
	assert.Equal(t, 1, docs)

	// an unknown plugin or version is an error and changes nothing
	assert.Error(t, client.UIApplicationPlugins.Install(ctx, appID, "missing"+suffix).Err)
	assert.Error(t, client.UIApplicationPlugins.Install(ctx, appID, plugin+"@9.9.9").Err)

	// delete (by contextPath) keeps the other remotes
	r = remotes(client.UIApplicationPlugins.Delete(ctx, appID, plugin+"-ctx"))
	assert.Equal(t, map[string]any{"legacy": []any{"LegacyModule"}}, r)

	// replace
	r = remotes(client.UIApplicationPlugins.Replace(ctx, appID, plugin+"@latest"))
	assert.Contains(t, r, key+"2.0.0")

	// invalid plugins are removed, more application fields are merged
	require.NoError(t, client.Applications.Update(ctx, appID, map[string]any{
		"config": map[string]any{"other": "kept", "remotes": map[string]any{key + "2.0.0": []string{"PlugModule"}, "gone@1.0.0": []string{"X"}}},
	}).Err)
	r = remotes(client.UIApplicationPlugins.Update(ctx, appID, applicationplugins.UpdateOptions{
		RemoveInvalid: true,
		Body:          map[string]any{"config": map[string]any{"excludedRemotes": []string{"x"}}},
	}))
	assert.NotContains(t, r, "gone@1.0.0")
	assert.Contains(t, r, key+"2.0.0")
	got := client.UIApplicationPlugins.List(ctx, appID)
	assert.Equal(t, `["x"]`, got.Data.Get("config.excludedRemotes").Raw)
}

func Test_UIApplicationPlugins_Plugins(t *testing.T) {
	testcore.SkipLive(t)
	app := applicationplugins.NewApplicationWithPlugins([]byte(`{"id":"99","config":{"remotes":{"b@2.0.0":["B"],"a@1.0.0":["A1","A2"],"legacy":["L"]}}}`))
	assert.Equal(t, "99", app.ID())
	assert.Equal(t, []applicationplugins.InstalledPlugin{
		{ContextPath: "a", Version: "1.0.0", Modules: []string{"A1", "A2"}},
		{ContextPath: "b", Version: "2.0.0", Modules: []string{"B"}},
	}, app.Plugins())
}
