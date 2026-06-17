package configuration

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
	if got := ByName("My Config"); got != "name:My Config" {
		t.Errorf("ByName = %q, want name:My Config", got)
	}
}

func TestScopeToConfiguration(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"", "$filter=(type eq 'c8y_ConfigurationDump')"},
		{"(name eq 'foo')", "$filter=(type eq 'c8y_ConfigurationDump' and (name eq 'foo'))"},
	}
	for _, tt := range tests {
		if got := ScopeToConfiguration(tt.filter); got != tt.want {
			t.Errorf("ScopeToConfiguration(%q) = %q, want %q", tt.filter, got, tt.want)
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
		_, _ = w.Write([]byte(`{"managedObjects":[{"id":"1","type":"c8y_ConfigurationDump"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{Query: ScopeToConfiguration("")})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotPath != "/inventory/managedObjects" {
		t.Errorf("path = %q, want /inventory/managedObjects", gotPath)
	}
	if !strings.Contains(gotQuery, "type eq 'c8y_ConfigurationDump'") {
		t.Errorf("q = %q, want type scoped", gotQuery)
	}
}

// TestGetByNameResolves verifies a name reference is resolved scoped to
// configuration files (type eq c8y_ConfigurationDump) and the resulting id is
// fetched.
func TestGetByNameResolves(t *testing.T) {
	var lookupQuery, getPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			lookupQuery = r.URL.Query().Get("q")
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"42","name":"myconfig","type":"c8y_ConfigurationDump"}]}`))
		case r.Method == http.MethodGet:
			getPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"42","name":"myconfig"}`))
		}
	})
	defer closeFn()

	res := svc.Get(context.Background(), ByName("myconfig"), GetOptions{})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if !strings.Contains(lookupQuery, "type eq 'c8y_ConfigurationDump'") || !strings.Contains(lookupQuery, "name eq 'myconfig'") {
		t.Errorf("lookup query = %q, want type+name scoped", lookupQuery)
	}
	if getPath != "/inventory/managedObjects/42" {
		t.Errorf("get path = %q, want .../42", getPath)
	}
}

// TestCreateRawInjectsTypeAndFragment verifies a map body gains the type and the
// c8y_Global fragment when absent.
func TestCreateRawInjectsTypeAndFragment(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	if res := svc.CreateRaw(context.Background(), map[string]any{"name": "c1"}); res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if body["type"] != TypeConfiguration {
		t.Errorf("type = %v, want %s", body["type"], TypeConfiguration)
	}
	if _, ok := body[FragmentGlobal]; !ok {
		t.Errorf("missing %s fragment: %v", FragmentGlobal, body)
	}
}

// TestCreateRawPassesRawBody verifies the CLI's json.RawMessage body is sent
// as-is (the body template, not the SDK, sets the type/fragment).
func TestCreateRawPassesRawBody(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	raw := json.RawMessage(`{"name":"c1","type":"c8y_ConfigurationDump","url":"https://ext/file"}`)
	if res := svc.CreateRaw(context.Background(), raw); res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if body["name"] != "c1" || body["url"] != "https://ext/file" {
		t.Errorf("raw body not passed through: %v", body)
	}
}

// TestCreateWithFileUploadsSetsURLAndLinks verifies the file path: the binary is
// uploaded, its url is written onto the configuration body, and the binary is
// linked back as a child addition of the new configuration.
func TestCreateWithFileUploadsSetsURLAndLinks(t *testing.T) {
	var moBody map[string]any
	var childPath string
	var childBody map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/binaries" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"b1","self":"` + "http://host/inventory/binaries/b1" + `"}`))
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&moBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"m1","name":"c1"}`))
		case strings.HasSuffix(r.URL.Path, "/childAdditions") && r.Method == http.MethodPost:
			childPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&childBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	defer closeFn()

	res := svc.Create(context.Background(), CreateOptions{
		Body: map[string]any{"name": "c1"},
		File: core.UploadFileOptions{Reader: strings.NewReader("{}"), Name: "c1.json"},
	})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if moBody["url"] != "http://host/inventory/binaries/b1" {
		t.Errorf("mo url = %v, want the uploaded binary self url", moBody["url"])
	}
	if moBody["type"] != TypeConfiguration {
		t.Errorf("mo type = %v, want %s", moBody["type"], TypeConfiguration)
	}
	if childPath != "/inventory/managedObjects/m1/childAdditions" {
		t.Errorf("child addition path = %q, want .../m1/childAdditions", childPath)
	}
	// The linked child is the uploaded binary (id b1).
	if ref, _ := childBody["managedObject"].(map[string]any); ref == nil || ref["id"] != "b1" {
		t.Errorf("child addition body = %v, want managedObject.id b1", childBody)
	}
}

// TestDeleteSendsForceCascade verifies the forceCascade query parameter is sent.
func TestDeleteSendsForceCascade(t *testing.T) {
	var gotMethod, gotPath, gotForce string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotForce = r.URL.Query().Get("forceCascade")
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), ByID("99"), DeleteOptions{ForceCascade: true})
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/inventory/managedObjects/99" {
		t.Errorf("delete = %s %s, want DELETE /inventory/managedObjects/99", gotMethod, gotPath)
	}
	if gotForce != "true" {
		t.Errorf("forceCascade = %q, want true", gotForce)
	}
}
