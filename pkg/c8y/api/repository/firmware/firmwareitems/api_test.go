package firmwareitems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestCreateRawPassesBodyThrough verifies CreateRaw POSTs the pre-built body to
// the managed objects endpoint unchanged.
func TestCreateRawPassesBodyThrough(t *testing.T) {
	var gotBody map[string]any
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1","type":"c8y_Firmware","name":"linux"}`))
	})
	defer closeFn()

	body := map[string]any{"type": "c8y_Firmware", "name": "linux", "c8y_Global": map[string]any{}}
	res := svc.CreateRaw(context.Background(), body)
	if res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/inventory/managedObjects" {
		t.Errorf("path = %q, want /inventory/managedObjects", gotPath)
	}
	if gotBody["name"] != "linux" || gotBody["type"] != "c8y_Firmware" {
		t.Errorf("body = %v, want name/type passthrough", gotBody)
	}
	if res.Data.ID() != "1" {
		t.Errorf("id = %q, want 1", res.Data.ID())
	}
}

// TestListForwardsQueryAndGetOptions verifies the extra raw Query filter is
// ANDed with the firmware type/name scoping and the GetOptions detail flags
// reach the request.
func TestListForwardsQueryAndGetOptions(t *testing.T) {
	var rawQuery, withChildren string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		rawQuery, _ = url.QueryUnescape(r.URL.RawQuery)
		withChildren = r.URL.Query().Get("withChildren")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	})
	defer closeFn()

	opt := ListOptions{Name: "linux", DeviceType: "c8y_Linux", Query: "has(c8y_Firmware)"}
	opt.WithChildren = true
	if res := svc.List(context.Background(), opt); res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	for _, want := range []string{"type eq 'c8y_Firmware'", "name eq 'linux'", "c8y_Filter.type eq 'c8y_Linux'", "has(c8y_Firmware)"} {
		if !strings.Contains(rawQuery, want) {
			t.Errorf("query = %q, want it to contain %q", rawQuery, want)
		}
	}
	if withChildren != "true" {
		t.Errorf("withChildren = %q, want true", withChildren)
	}
}
