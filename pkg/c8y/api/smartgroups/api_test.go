package smartgroups

import (
	"context"
	"encoding/json"
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

func TestRefConstructors(t *testing.T) {
	if got := ByID("123"); got != "123" {
		t.Errorf("ByID = %q, want 123", got)
	}
	if got := ByName("My Group"); got != "name:My Group" {
		t.Errorf("ByName = %q, want name:My Group", got)
	}
}

func TestScopeToSmartGroups(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"", "$filter=(type eq 'c8y_DynamicGroup')"},
		{"(name eq 'foo')", "$filter=(type eq 'c8y_DynamicGroup' and (name eq 'foo'))"},
	}
	for _, tt := range tests {
		if got := ScopeToSmartGroups(tt.filter); got != tt.want {
			t.Errorf("ScopeToSmartGroups(%q) = %q, want %q", tt.filter, got, tt.want)
		}
	}
}

// TestGetByNameResolves verifies a name reference is resolved scoped to smart
// groups (type eq c8y_DynamicGroup) and the resulting id is fetched.
func TestGetByNameResolves(t *testing.T) {
	var lookupQuery, getPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			lookupQuery = r.URL.Query().Get("q")
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"42","name":"mygroup","type":"c8y_DynamicGroup"}]}`))
		case r.Method == http.MethodGet:
			getPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"42","name":"mygroup"}`))
		}
	})
	defer closeFn()

	res := svc.Get(context.Background(), ByName("mygroup"), GetOptions{})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if !strings.Contains(lookupQuery, "type eq 'c8y_DynamicGroup'") || !strings.Contains(lookupQuery, "name eq 'mygroup'") {
		t.Errorf("lookup query = %q, want type+name scoped", lookupQuery)
	}
	if getPath != "/inventory/managedObjects/42" {
		t.Errorf("get path = %q, want .../42", getPath)
	}
}

// TestCreateInjectsTypeAndFragment verifies a map body gains the type and
// fragment when absent.
func TestCreateInjectsTypeAndFragment(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	if res := svc.Create(context.Background(), map[string]any{"name": "g1"}); res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if body["type"] != TypeDynamicGroup {
		t.Errorf("type = %v, want %s", body["type"], TypeDynamicGroup)
	}
	if _, ok := body[FragmentIsDynamicGroup]; !ok {
		t.Errorf("missing %s fragment: %v", FragmentIsDynamicGroup, body)
	}
}
