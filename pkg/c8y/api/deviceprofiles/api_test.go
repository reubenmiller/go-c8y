package deviceprofiles

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
	if got := ByName("My Profile"); got != "name:My Profile" {
		t.Errorf("ByName = %q, want name:My Profile", got)
	}
}

func TestScopeToDeviceProfiles(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"", "$filter=(type eq 'c8y_Profile')"},
		{"(name eq 'foo')", "$filter=(type eq 'c8y_Profile' and (name eq 'foo'))"},
	}
	for _, tt := range tests {
		if got := ScopeToDeviceProfiles(tt.filter); got != tt.want {
			t.Errorf("ScopeToDeviceProfiles(%q) = %q, want %q", tt.filter, got, tt.want)
		}
	}
}

// TestListScopesToType verifies the q param passes through to the managed
// object collection endpoint unchanged.
func TestListScopesToType(t *testing.T) {
	var gotQuery, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[{"id":"1","type":"c8y_Profile"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{Query: ScopeToDeviceProfiles("")})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotPath != "/inventory/managedObjects" {
		t.Errorf("path = %q, want /inventory/managedObjects", gotPath)
	}
	if !strings.Contains(gotQuery, "type eq 'c8y_Profile'") {
		t.Errorf("q = %q, want type scoped", gotQuery)
	}
}

// TestGetByNameResolves verifies a name reference is resolved scoped to device
// profiles (type eq c8y_Profile) and the resulting id is fetched.
func TestGetByNameResolves(t *testing.T) {
	var lookupQuery, getPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			lookupQuery = r.URL.Query().Get("q")
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"42","name":"myprofile","type":"c8y_Profile"}]}`))
		case r.Method == http.MethodGet:
			getPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"42","name":"myprofile"}`))
		}
	})
	defer closeFn()

	res := svc.Get(context.Background(), ByName("myprofile"), GetOptions{})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if !strings.Contains(lookupQuery, "type eq 'c8y_Profile'") || !strings.Contains(lookupQuery, "name eq 'myprofile'") {
		t.Errorf("lookup query = %q, want type+name scoped", lookupQuery)
	}
	if getPath != "/inventory/managedObjects/42" {
		t.Errorf("get path = %q, want .../42", getPath)
	}
}

// TestCreateInjectsTypeAndFragments verifies a map body gains the type and the
// c8y_DeviceProfile / c8y_Filter fragments when absent.
func TestCreateInjectsTypeAndFragments(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	if res := svc.Create(context.Background(), map[string]any{"name": "p1"}); res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if body["type"] != TypeProfile {
		t.Errorf("type = %v, want %s", body["type"], TypeProfile)
	}
	if _, ok := body[FragmentDeviceProfile]; !ok {
		t.Errorf("missing %s fragment: %v", FragmentDeviceProfile, body)
	}
	if _, ok := body[FragmentFilter]; !ok {
		t.Errorf("missing %s fragment: %v", FragmentFilter, body)
	}
}

// TestCreatePreservesExistingFragments verifies an existing c8y_Filter (e.g. a
// deviceType filter) is not overwritten by the injected empty fragment.
func TestCreatePreservesExistingFragments(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	in := map[string]any{
		"name":       "p1",
		"c8y_Filter": map[string]any{"type": "myType"},
	}
	if res := svc.Create(context.Background(), in); res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	filter, ok := body[FragmentFilter].(map[string]any)
	if !ok || filter["type"] != "myType" {
		t.Errorf("c8y_Filter = %v, want preserved {type: myType}", body[FragmentFilter])
	}
}
