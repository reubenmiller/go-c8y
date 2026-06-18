package bulkoperations

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

// TestList verifies the bulk-operation collection is plucked into items.
func TestList(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bulkOperations":[{"id":"1"},{"id":"2"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != ApiBulkOperations {
		t.Errorf("path = %q, want %q", path, ApiBulkOperations)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Errorf("ids = %v, want [1 2]", ids)
	}
}

// TestGet verifies the bulk-operation id is sent on the single-item path.
func TestGet(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "42")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if method != http.MethodGet {
		t.Errorf("method = %q, want GET", method)
	}
	if path != "/devicecontrol/bulkoperations/42" {
		t.Errorf("path = %q, want .../bulkoperations/42", path)
	}
}

// TestCreate verifies the request body is passed through unchanged.
func TestCreate(t *testing.T) {
	var body map[string]any
	var method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	if res := svc.Create(context.Background(), map[string]any{"groupId": "555"}); res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %q, want POST", method)
	}
	if body["groupId"] != "555" {
		t.Errorf("groupId = %v, want 555", body["groupId"])
	}
}

// TestUpdate verifies the id is sent on the path and the body passes through.
func TestUpdate(t *testing.T) {
	var path, method string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42"}`))
	})
	defer closeFn()

	if res := svc.Update(context.Background(), "42", map[string]any{"creationRamp": 15}); res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut {
		t.Errorf("method = %q, want PUT", method)
	}
	if path != "/devicecontrol/bulkoperations/42" {
		t.Errorf("path = %q, want .../bulkoperations/42", path)
	}
	if body["creationRamp"] != float64(15) {
		t.Errorf("creationRamp = %v, want 15", body["creationRamp"])
	}
}

// TestDelete verifies a 204 yields no error and the id is on the path.
func TestDelete(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	if res := svc.Delete(context.Background(), "42"); res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", method)
	}
	if path != "/devicecontrol/bulkoperations/42" {
		t.Errorf("path = %q, want .../bulkoperations/42", path)
	}
}

// TestListOperations verifies operations belonging to a bulk operation are
// fetched from the operations collection, scoped by the bulkOperationId query
// parameter, with the secondary filters forwarded and the "operations"
// collection plucked into items.
func TestListOperations(t *testing.T) {
	var path string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"operations":[{"id":"o1"},{"id":"o2"}]}`))
	})
	defer closeFn()

	res := svc.ListOperations(context.Background(), ListOperationsOptions{
		BulkOperationID: "10",
		Status:          "PENDING",
		Revert:          true,
	})
	if res.Err != nil {
		t.Fatalf("ListOperations: %v", res.Err)
	}
	if path != ApiOperations {
		t.Errorf("path = %q, want %q", path, ApiOperations)
	}
	if got := query.Get("bulkOperationId"); got != "10" {
		t.Errorf("bulkOperationId = %q, want 10", got)
	}
	if got := query.Get("status"); got != "PENDING" {
		t.Errorf("status = %q, want PENDING", got)
	}
	if got := query.Get("revert"); got != "true" {
		t.Errorf("revert = %q, want true", got)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.Get("id").String())
	}
	if len(ids) != 2 || ids[0] != "o1" || ids[1] != "o2" {
		t.Errorf("ids = %v, want [o1 o2]", ids)
	}
}
