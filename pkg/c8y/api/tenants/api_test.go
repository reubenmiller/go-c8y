package tenants

import (
	"context"
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

// TestList verifies the tenants collection is plucked into items and the
// company filter is serialized as a query parameter.
func TestList(t *testing.T) {
	var path, query string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tenants":[{"id":"t1"},{"id":"t2"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{Company: "acme"})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/tenant/tenants" {
		t.Errorf("path = %q, want /tenant/tenants", path)
	}
	if !strings.Contains(query, "company=acme") {
		t.Errorf("query = %q, want company=acme", query)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "t1" || ids[1] != "t2" {
		t.Errorf("ids = %v, want [t1 t2]", ids)
	}
}

// TestUpdate verifies the body is passed through to PUT /tenant/tenants/{id}.
// This backs the enable/disable (suspend/activate) commands, which set a static
// status field.
func TestUpdate(t *testing.T) {
	var path, method, body string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"t1","status":"SUSPENDED"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), "t1", map[string]any{"status": "SUSPENDED"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut {
		t.Errorf("method = %q, want PUT", method)
	}
	if path != "/tenant/tenants/t1" {
		t.Errorf("path = %q, want /tenant/tenants/t1", path)
	}
	if !strings.Contains(body, `"status":"SUSPENDED"`) {
		t.Errorf("body = %q, want status SUSPENDED", body)
	}
}

// TestListApplicationReferences verifies the references collection is plucked
// and the tenant id is set on the path.
func TestListApplicationReferences(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"references":[{"application":{"id":"a1"}},{"application":{"id":"a2"}}]}`))
	})
	defer closeFn()

	res := svc.ListApplicationReferences(context.Background(), "t1", ListApplicationReferencesOptions{})
	if res.Err != nil {
		t.Fatalf("ListApplicationReferences: %v", res.Err)
	}
	if path != "/tenant/tenants/t1/applications" {
		t.Errorf("path = %q, want /tenant/tenants/t1/applications", path)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.Application().ID())
	}
	if len(ids) != 2 || ids[0] != "a1" || ids[1] != "a2" {
		t.Errorf("application ids = %v, want [a1 a2]", ids)
	}
}

// TestSubscribeApplication verifies the application reference body is POSTed to
// the tenant's applications endpoint and the reference is returned as-is.
func TestSubscribeApplication(t *testing.T) {
	var path, method, body string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"self":"https://example/tenant/tenants/t1/applications/a1","application":{"id":"a1"}}`))
	})
	defer closeFn()

	res := svc.SubscribeApplication(context.Background(), "t1", map[string]any{"application": map[string]any{"id": "a1"}})
	if res.Err != nil {
		t.Fatalf("SubscribeApplication: %v", res.Err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %q, want POST", method)
	}
	if path != "/tenant/tenants/t1/applications" {
		t.Errorf("path = %q, want /tenant/tenants/t1/applications", path)
	}
	if !strings.Contains(body, `"id":"a1"`) {
		t.Errorf("body = %q, want application.id a1", body)
	}
	ref, err := res.First()
	if err != nil {
		t.Fatalf("First: %v", err)
	}
	if got := ref.Application().ID(); got != "a1" {
		t.Errorf("reference application id = %q, want a1", got)
	}
}

// TestUnsubscribeApplication verifies the tenant and application ids are set on
// the DELETE path.
func TestUnsubscribeApplication(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.UnsubscribeApplication(context.Background(), "t1", "a1")
	if res.Err != nil {
		t.Fatalf("UnsubscribeApplication: %v", res.Err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", method)
	}
	if path != "/tenant/tenants/t1/applications/a1" {
		t.Errorf("path = %q, want /tenant/tenants/t1/applications/a1", path)
	}
}

// TestTFA verifies the two-factor authentication get/update paths and that the
// update body is passed through.
func TestTFA(t *testing.T) {
	var getPath, updatePath, updateBody, updateMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			getPath = r.URL.Path
			_, _ = w.Write([]byte(`{"enabled":true,"strategy":"TOTP"}`))
		case http.MethodPut:
			updatePath = r.URL.Path
			updateMethod = r.Method
			b, _ := io.ReadAll(r.Body)
			updateBody = string(b)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	defer closeFn()

	if res := svc.GetTFA(context.Background(), "t1"); res.Err != nil {
		t.Fatalf("GetTFA: %v", res.Err)
	}
	if getPath != "/tenant/tenants/t1/tfa" {
		t.Errorf("get path = %q, want /tenant/tenants/t1/tfa", getPath)
	}
	if res := svc.UpdateTFA(context.Background(), "t1", map[string]any{"strategy": "SMS"}); res.Err != nil {
		t.Fatalf("UpdateTFA: %v", res.Err)
	}
	if updateMethod != http.MethodPut {
		t.Errorf("update method = %q, want PUT", updateMethod)
	}
	if updatePath != "/tenant/tenants/t1/tfa" {
		t.Errorf("update path = %q, want /tenant/tenants/t1/tfa", updatePath)
	}
	if !strings.Contains(updateBody, `"strategy":"SMS"`) {
		t.Errorf("update body = %q, want strategy SMS", updateBody)
	}
}
