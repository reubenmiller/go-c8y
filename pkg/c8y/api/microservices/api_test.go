package microservices

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestList verifies microservices are listed via the applications endpoint with
// the MICROSERVICE type filter, and the applications collection is plucked.
func TestList(t *testing.T) {
	var path, typeParam string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		typeParam = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"applications":[{"id":"1","name":"app-a"},{"id":"2","name":"app-b"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/application/applications" {
		t.Errorf("path = %q, want /application/applications", path)
	}
	if typeParam != "MICROSERVICE" {
		t.Errorf("type = %q, want MICROSERVICE", typeParam)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Errorf("ids = %v, want [1 2]", ids)
	}
}

// TestGetByID verifies a plain id is fetched directly (no name lookup).
func TestGetByID(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","name":"my-svc"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if path != "/application/applications/123" {
		t.Errorf("path = %q, want .../123", path)
	}
	if res.Data.Name() != "my-svc" {
		t.Errorf("name = %q, want my-svc", res.Data.Name())
	}
}

// TestUpdate verifies the body is passed through on a PUT to the application.
func TestUpdate(t *testing.T) {
	var method, path string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","availability":"MARKET"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), "123", map[string]any{"availability": "MARKET"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut || path != "/application/applications/123" {
		t.Errorf("got %s %s, want PUT .../123", method, path)
	}
	if body["availability"] != "MARKET" {
		t.Errorf("body availability = %v, want MARKET", body["availability"])
	}
}

// TestDeleteForce verifies the force query parameter is sent (--unsubscribeAll).
func TestDeleteForce(t *testing.T) {
	var method, path, force string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		force = r.URL.Query().Get("force")
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), "123", DeleteOptions{Force: true})
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if method != http.MethodDelete || path != "/application/applications/123" {
		t.Errorf("got %s %s, want DELETE .../123", method, path)
	}
	if force != "true" {
		t.Errorf("force = %q, want true", force)
	}
}

// TestEnable verifies the subscription POST hits the tenant applications
// endpoint with the application reference body and returns the reference as-is.
func TestEnable(t *testing.T) {
	var method, path string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"self":"https://x/ref","application":{"id":"123","name":"my-svc"}}`))
	})
	defer closeFn()

	res := svc.Enable(context.Background(), "t100", map[string]any{"application": map[string]any{"id": "123"}})
	if res.Err != nil {
		t.Fatalf("Enable: %v", res.Err)
	}
	if method != http.MethodPost || path != "/tenant/tenants/t100/applications" {
		t.Errorf("got %s %s, want POST /tenant/tenants/t100/applications", method, path)
	}
	app, _ := body["application"].(map[string]any)
	if app == nil || app["id"] != "123" {
		t.Errorf("body application.id = %v, want 123", body["application"])
	}
	// Reference is returned as-is (self + application), not plucked.
	if res.Data.Self() != "https://x/ref" || res.Data.Application().ID() != "123" {
		t.Errorf("reference = %s, want self+application", res.Data.Raw())
	}
}

// TestUnsubscribe verifies disable hits the tenant application delete path.
func TestUnsubscribe(t *testing.T) {
	var method, path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Unsubscribe(context.Background(), "t100", "123")
	if res.Err != nil {
		t.Fatalf("Unsubscribe: %v", res.Err)
	}
	if method != http.MethodDelete || path != "/tenant/tenants/t100/applications/123" {
		t.Errorf("got %s %s, want DELETE /tenant/tenants/t100/applications/123", method, path)
	}
}

// TestGetStatus verifies the inventory query filters by the c8y_Application_<id>
// type and plucks managedObjects.
func TestGetStatus(t *testing.T) {
	var path, typeParam string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		typeParam = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[{"id":"9","type":"c8y_Application_123"}]}`))
	})
	defer closeFn()

	res := svc.GetStatus(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("GetStatus: %v", res.Err)
	}
	if path != "/inventory/managedObjects" {
		t.Errorf("path = %q, want /inventory/managedObjects", path)
	}
	if typeParam != "c8y_Application_123" {
		t.Errorf("type = %q, want c8y_Application_123", typeParam)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 1 || ids[0] != "9" {
		t.Errorf("ids = %v, want [9]", ids)
	}
}

// TestUpload verifies the binary is sent as a multipart POST to the binaries
// sub-resource (the FilePath form must stream the file).
func TestUpload(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "helloworld.zip")
	if err := os.WriteFile(file, []byte("PK\x03\x04 dummy zip"), 0o600); err != nil {
		t.Fatal(err)
	}

	var method, path, contentType string
	var gotFile bool
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		contentType = r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			if mr, err := r.MultipartReader(); err == nil {
				for {
					part, err := mr.NextPart()
					if err != nil {
						break
					}
					if part.FormName() == "file" {
						b, _ := io.ReadAll(part)
						gotFile = len(b) > 0
					}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"bin1","name":"helloworld.zip"}`))
	})
	defer closeFn()

	res := svc.Upload(context.Background(), "123", UploadFileOptions{FilePath: file})
	if res.Err != nil {
		t.Fatalf("Upload: %v", res.Err)
	}
	if method != http.MethodPost || path != "/application/applications/123/binaries" {
		t.Errorf("got %s %s, want POST .../123/binaries", method, path)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Errorf("content-type = %q, want multipart/form-data", contentType)
	}
	if !gotFile {
		t.Error("expected a non-empty file part")
	}
}

// TestBootstrapUserGet verifies the bootstrap-user sub-resource path.
func TestBootstrapUserGet(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"servicebootstrap_my-svc","password":"secret","tenant":"t100"}`))
	})
	defer closeFn()

	res := svc.BootstrapUser.Get(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("BootstrapUser.Get: %v", res.Err)
	}
	if path != "/application/applications/123/bootstrapUser" {
		t.Errorf("path = %q, want .../123/bootstrapUser", path)
	}
}
