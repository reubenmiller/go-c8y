package auditrecords

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestGet verifies the id is substituted into the audit record path.
func TestGet(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","type":"Inventory"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if method != http.MethodGet {
		t.Errorf("method = %q, want GET", method)
	}
	if path != "/audit/auditRecords/123" {
		t.Errorf("path = %q, want /audit/auditRecords/123", path)
	}
	if res.Data.ID() != "123" {
		t.Errorf("id = %q, want 123", res.Data.ID())
	}
}

// TestCreate verifies the body is POSTed as-is to the collection endpoint.
func TestCreate(t *testing.T) {
	var path, method string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	res := svc.Create(context.Background(), map[string]any{
		"type":     "Inventory",
		"activity": "Managed Object updated",
		"text":     "details",
		"source":   map[string]any{"id": "12345"},
	})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %q, want POST", method)
	}
	if path != "/audit/auditRecords" {
		t.Errorf("path = %q, want /audit/auditRecords", path)
	}
	if body["activity"] != "Managed Object updated" {
		t.Errorf("body activity = %v, want passthrough", body["activity"])
	}
	if src, _ := body["source"].(map[string]any); src["id"] != "12345" {
		t.Errorf("body source.id = %v, want 12345", body["source"])
	}
}

// TestListPluckAndRevert verifies the collection is plucked from the
// auditRecords property and that the revert overlay extraField (absent from the
// OpenAPI spec, supported by the server) is emitted as a query parameter.
func TestListPluckAndRevert(t *testing.T) {
	var path string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auditRecords":[{"id":"a1"},{"id":"a2"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{
		Source: "12345",
		Type:   "Inventory",
		User:   "admin",
		Revert: true,
	})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/audit/auditRecords" {
		t.Errorf("path = %q, want /audit/auditRecords", path)
	}
	if query.Get("revert") != "true" {
		t.Errorf("revert = %q, want true", query.Get("revert"))
	}
	if query.Get("source") != "12345" || query.Get("type") != "Inventory" || query.Get("user") != "admin" {
		t.Errorf("query = %v, want source/type/user set", query)
	}

	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "a1" || ids[1] != "a2" {
		t.Errorf("ids = %v, want [a1 a2]", ids)
	}
}
