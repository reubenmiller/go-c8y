package applicationplugins_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
)

const (
	appJSON    = `{"id":"11","name":"cockpit","type":"HOSTED","contextPath":"cockpit","config":{"remotes":{}}}`
	pluginJSON = `{"id":"12","name":"my-plugin","type":"HOSTED","contextPath":"my-plugin","manifest":{"remotes":{"my-plugin":["Mod"]}},"applicationVersions":[{"version":"1.0.0","tags":["latest"]}]}`
)

// pluginServer answers the application 11, the plugin 12 (by id and in the lists)
// and records the requests.
func pluginServer(t *testing.T) (*api.Client, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tenant/currentTenant" { // the client's own tenant discovery
			mu.Lock()
			requests = append(requests, r.Method+" "+r.URL.RequestURI())
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/application/applications/11":
			_, _ = w.Write([]byte(appJSON))
		case "/application/applications/12":
			_, _ = w.Write([]byte(pluginJSON))
		case "/application/applications/99":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"general/Not Found","message":"not found"}`))
		case "/application/applications":
			_, _ = w.Write([]byte(`{"applications":[` + pluginJSON + `],"statistics":{"pageSize":2000,"currentPage":1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	client := api.NewClient(api.ClientOptions{BaseURL: srv.URL})
	return client, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requests...)
	}
}

// TestPluginById checks that a plugin given by id is fetched by id, with no
// listing of the hosted and shared applications, as v2.59.0 did.
func TestPluginById(t *testing.T) {
	for name, call := range map[string]func(c *api.Client) error{
		"install": func(c *api.Client) error {
			return c.UIApplicationPlugins.Install(api.WithDryRun(context.Background(), true), "11", "12").Err
		},
		"delete": func(c *api.Client) error {
			return c.UIApplicationPlugins.Delete(api.WithDryRun(context.Background(), true), "11", "12").Err
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, requests := pluginServer(t)
			if err := call(client); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			got := strings.Join(requests(), "\n")
			want := "GET /application/applications/11\nGET /application/applications/12"
			if got != want {
				t.Errorf("requests =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// TestPluginByName checks that a plugin given by name is looked up in the
// catalog (the hosted and the shared applications).
func TestPluginByName(t *testing.T) {
	client, requests := pluginServer(t)
	if err := client.UIApplicationPlugins.Install(api.WithDryRun(context.Background(), true), "11", "my-plugin").Err; err != nil {
		t.Fatal(err)
	}
	got := requests()
	if len(got) != 3 || got[0] != "GET /application/applications/11" || !strings.Contains(got[1], "type=HOSTED") || !strings.Contains(got[2], "availability=SHARED") {
		t.Errorf("requests = %v", got)
	}
}

// TestPluginNumericNameFallsBackToCatalog checks that a reference which looks
// like an id but is no plugin id is looked up by name.
func TestPluginNumericNameFallsBackToCatalog(t *testing.T) {
	client, requests := pluginServer(t)
	err := client.UIApplicationPlugins.Install(api.WithDryRun(context.Background(), true), "11", "99").Err
	if err == nil || !strings.Contains(err.Error(), "plugin does not exist. name=99") {
		t.Fatalf("error = %v", err)
	}
	if got := requests(); len(got) != 4 || got[1] != "GET /application/applications/99" {
		t.Errorf("requests = %v", got)
	}
}
