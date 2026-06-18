package remoteaccess_configurations

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestList verifies the device-scoped collection path and that a plain JSON
// array body (as the remoteaccess microservice returns) is plucked into items.
func TestList(t *testing.T) {
	var gotMethod, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","name":"telnet","protocol":"TELNET"},{"id":"2","name":"webssh","protocol":"SSH"}]`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{ManagedObjectID: "d1"})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/service/remoteaccess/devices/d1/configurations" {
		t.Errorf("path = %q, want /service/remoteaccess/devices/d1/configurations", gotPath)
	}
	var n int
	for cfg, err := range res.Items() {
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		n++
		if cfg.ID() == "" || cfg.Name() == "" {
			t.Errorf("item %d missing id/name: %s", n, cfg.Raw())
		}
	}
	if n != 2 {
		t.Errorf("item count = %d, want 2", n)
	}
}

func TestGet(t *testing.T) {
	var gotMethod, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"7","name":"telnet"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{ManagedObjectID: "d1", ConfigurationID: "7"})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/service/remoteaccess/devices/d1/configurations/7" {
		t.Errorf("path = %q, want .../configurations/7", gotPath)
	}
	if got := res.Data.ID(); got != "7" {
		t.Errorf("id = %q, want 7", got)
	}
}

func TestCreatePassesBodyThrough(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"9","name":"telnet"}`))
	})
	defer closeFn()

	body := json.RawMessage(`{"name":"telnet","hostname":"127.0.0.1","port":23,"protocol":"TELNET"}`)
	res := svc.Create(context.Background(), CreateOptions{ManagedObjectID: "d1", Body: body})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/service/remoteaccess/devices/d1/configurations" {
		t.Errorf("path = %q, want .../configurations", gotPath)
	}
	if !strings.Contains(gotBody, `"protocol":"TELNET"`) || !strings.Contains(gotBody, `"name":"telnet"`) {
		t.Errorf("body not passed through verbatim: %q", gotBody)
	}
}

func TestUpdatePassesBodyThrough(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"7","name":"renamed"}`))
	})
	defer closeFn()

	body := json.RawMessage(`{"name":"renamed"}`)
	res := svc.Update(context.Background(), UpdateOptions{ManagedObjectID: "d1", ConfigurationID: "7", Body: body})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/service/remoteaccess/devices/d1/configurations/7" {
		t.Errorf("path = %q, want .../configurations/7", gotPath)
	}
	if !strings.Contains(gotBody, `"name":"renamed"`) {
		t.Errorf("body not passed through: %q", gotBody)
	}
}

func TestDelete(t *testing.T) {
	var gotMethod, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), DeleteOptions{ManagedObjectID: "d1", ConfigurationID: "7"})
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/service/remoteaccess/devices/d1/configurations/7" {
		t.Errorf("path = %q, want .../configurations/7", gotPath)
	}
}

// TestResolveIDPlainID verifies a plain id passes through with no HTTP call.
func TestResolveIDPlainID(t *testing.T) {
	var called bool
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	defer closeFn()

	got, err := svc.ResolveID(context.Background(), "d1", "42")
	if err != nil {
		t.Fatalf("ResolveID: %v", err)
	}
	if got != "42" {
		t.Errorf("id = %q, want 42", got)
	}
	if called {
		t.Error("plain id should not trigger a list lookup")
	}
}

// TestResolveIDByName verifies a name reference lists the device's
// configurations and returns the matching id (case-insensitive).
func TestResolveIDByName(t *testing.T) {
	var listPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		listPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","name":"telnet"},{"id":"2","name":"webssh"}]`))
	})
	defer closeFn()

	got, err := svc.ResolveID(context.Background(), "d1", "name:WebSSH")
	if err != nil {
		t.Fatalf("ResolveID: %v", err)
	}
	if got != "2" {
		t.Errorf("id = %q, want 2", got)
	}
	if listPath != "/service/remoteaccess/devices/d1/configurations" {
		t.Errorf("list path = %q, want device-scoped configurations", listPath)
	}

	if _, err := svc.ResolveID(context.Background(), "d1", "name:missing"); err == nil {
		t.Error("expected error for an unknown configuration name")
	}
}
