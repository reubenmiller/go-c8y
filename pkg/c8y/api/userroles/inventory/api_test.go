package inventoryroles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestList verifies the collection endpoint path and that the "roles" result
// property is plucked.
func TestList(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"roles":[{"id":1,"name":"read"}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotPath != "/user/inventoryroles" {
		t.Errorf("path = %q, want /user/inventoryroles", gotPath)
	}
	first, err := res.First()
	if err != nil {
		t.Fatalf("First: %v", err)
	}
	if first.Get("name").String() != "read" {
		t.Errorf("role name = %q, want read (roles plucked)", first.Get("name").String())
	}
}

// TestGet verifies the single-item endpoint formats the numeric id into the path.
func TestGet(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"name":"read"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), 42)
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotPath != "/user/inventoryroles/42" {
		t.Errorf("path = %q, want /user/inventoryroles/42", gotPath)
	}
}
