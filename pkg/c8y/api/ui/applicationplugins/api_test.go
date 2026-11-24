package applicationplugins

import (
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
)

func TestResolveVersion(t *testing.T) {
	plugin := jsonmodels.NewUIPlugin([]byte(`{
		"name": "My Plugin", "contextPath": "my-plugin",
		"manifest": {"remotes": {"my-plugin": ["ModA", "ModB"]}},
		"applicationVersions": [
			{"version": "1.0.0", "tags": ["stable"]},
			{"version": "2.0.0", "tags": ["latest"]}
		]
	}`))
	for _, tt := range []struct{ ref, want string }{
		{"", "2.0.0"},
		{"latest", "2.0.0"},
		{"stable", "1.0.0"},
		{"1.0.0", "1.0.0"},
	} {
		got, err := resolveVersion(plugin, tt.ref)
		if err != nil {
			t.Fatalf("%q: %v", tt.ref, err)
		}
		if got.Version != tt.want || got.ContextPath != "my-plugin" || len(got.Modules) != 2 {
			t.Errorf("%q: got %+v, want version %s", tt.ref, got, tt.want)
		}
	}
	if _, err := resolveVersion(plugin, "9.9.9"); err == nil {
		t.Error("an unknown version is an error")
	}
}

func TestApplicationBody(t *testing.T) {
	app := NewApplicationWithPlugins([]byte(`{"id":"1","config":{
		"remotes": {"old@1.0.0": ["Old"], "legacy": ["L"]},
		"excludedRemotes": ["a"],
		"nested": {"keep": 1, "change": 1}
	}}`))
	installed := map[string]InstalledPlugin{
		"new": {ContextPath: "new", Version: "2.0.0", Modules: []string{"New"}},
	}
	extra := map[string]any{"config": map[string]any{"nested": map[string]any{"change": 2}}, "name": "renamed"}

	body, err := applicationBody(app, installed, extra)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(body)
	want := `{"config":{"excludedRemotes":["a"],"nested":{"change":2,"keep":1},"remotes":{"legacy":["L"],"new@2.0.0":["New"]}},"name":"renamed"}`
	if string(b) != want {
		t.Errorf("body =\n%s\nwant\n%s", b, want)
	}
}

func TestApplicationBody_NoConfig(t *testing.T) {
	body, err := applicationBody(NewApplicationWithPlugins([]byte(`{"id":"1"}`)), map[string]InstalledPlugin{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(body)
	if string(b) != `{"config":{"remotes":{}}}` {
		t.Errorf("body = %s", b)
	}
}
