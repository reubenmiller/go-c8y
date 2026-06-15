package currenttenant

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
	svc := NewService(&core.Service{
		Client: resty.New().SetBaseURL(ts.URL),
	})
	return svc, ts.Close
}

func TestListApplications(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name":"t12345",
			"applications":{"references":[
				{"application":{"id":"1","name":"cockpit"}},
				{"application":{"id":"2","name":"devicemanagement"}}
			]}
		}`))
	})
	defer closeFn()

	res := svc.ListApplications(context.Background())
	if res.Err != nil {
		t.Fatalf("ListApplications: %v", res.Err)
	}
	if gotPath != ApiTenant {
		t.Errorf("path = %q, want %q", gotPath, ApiTenant)
	}
	var names []string
	for app, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		names = append(names, app.Name())
	}
	if len(names) != 2 || names[0] != "cockpit" || names[1] != "devicemanagement" {
		t.Errorf("apps = %v, want [cockpit devicemanagement]", names)
	}
}

func TestGetVersion(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"category":"system","key":"version","value":"1020.0.1"}`))
	})
	defer closeFn()

	res := svc.GetVersion(context.Background())
	if res.Err != nil {
		t.Fatalf("GetVersion: %v", res.Err)
	}
	if gotPath != ApiSystemVersion {
		t.Errorf("path = %q, want %q", gotPath, ApiSystemVersion)
	}
	if v := res.Data.Get("value").String(); v != "1020.0.1" {
		t.Errorf("version = %q, want 1020.0.1", v)
	}
}
