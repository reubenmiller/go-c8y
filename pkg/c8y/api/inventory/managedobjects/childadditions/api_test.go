package childadditions

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestListPlucksReferences verifies the collection is read from the
// references.#.managedObject path of a managedObjectReferenceCollection.
func TestListPlucksReferences(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"references":[{"managedObject":{"id":"11"}},{"managedObject":{"id":"12"}}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), "100", ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/inventory/managedObjects/100/childAdditions" {
		t.Errorf("list path = %q", path)
	}
	var ids []string
	for item := range op.Iter(res) {
		ids = append(ids, item.ID())
	}
	if strings.Join(ids, ",") != "11,12" {
		t.Errorf("plucked ids = %v, want [11 12]", ids)
	}
}

// TestGetReturnsNestedManagedObject verifies GET …/{child} returns the nested
// managedObject of the reference, not the reference envelope.
func TestGetReturnsNestedManagedObject(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"self":"x","managedObject":{"id":"77","name":"child"}}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "100", "77")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if path != "/inventory/managedObjects/100/childAdditions/77" {
		t.Errorf("get path = %q", path)
	}
	if res.Data.ID() != "77" {
		t.Errorf("id = %q, want 77 (nested managedObject)", res.Data.ID())
	}
}

// TestCreatePostsManagedObject verifies a new child is created with the
// managedObject content-type and the body passed through.
func TestCreatePostsManagedObject(t *testing.T) {
	var path, method, ct string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method, ct = r.URL.Path, r.Method, r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"5"}`))
	})
	defer closeFn()

	res := svc.Create(context.Background(), "100", map[string]any{"name": "new"})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if method != http.MethodPost || path != "/inventory/managedObjects/100/childAdditions" {
		t.Errorf("create = %s %s", method, path)
	}
	if !strings.Contains(ct, "managedobject") {
		t.Errorf("content-type = %q, want managedobject", ct)
	}
	if body["name"] != "new" {
		t.Errorf("body = %v", body)
	}
}

// TestAssignPostsReference verifies an existing child is linked via a
// managedObjectReference body ({"managedObject":{"id":…}}).
func TestAssignPostsReference(t *testing.T) {
	var path, method string
	var body struct {
		ManagedObject struct {
			ID string `json:"id"`
		} `json:"managedObject"`
	}
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
	})
	defer closeFn()

	if res := svc.Assign(context.Background(), "100", "55"); res.Err != nil {
		t.Fatalf("Assign: %v", res.Err)
	}
	if method != http.MethodPost || path != "/inventory/managedObjects/100/childAdditions" {
		t.Errorf("assign = %s %s", method, path)
	}
	if body.ManagedObject.ID != "55" {
		t.Errorf("assign body managedObject.id = %q, want 55", body.ManagedObject.ID)
	}
}

// TestUnassignDeletesByPath verifies a single reference is removed via
// DELETE …/{child} with no request body (not a collection delete).
func TestUnassignDeletesByPath(t *testing.T) {
	var path, method string
	var bodyLen int
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		bodyLen = len(b)
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	if res := svc.Unassign(context.Background(), "100", "55"); res.Err != nil {
		t.Fatalf("Unassign: %v", res.Err)
	}
	if method != http.MethodDelete || path != "/inventory/managedObjects/100/childAdditions/55" {
		t.Errorf("unassign = %s %s, want DELETE .../childAdditions/55", method, path)
	}
	if bodyLen != 0 {
		t.Errorf("unassign sent a %d-byte body, want path-based delete with no body", bodyLen)
	}
}
