package currentapplication

import (
	"context"
	"encoding/json"
	"io"
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

func TestGet(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","name":"my-app","type":"MICROSERVICE"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background())
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != ApiApplication {
		t.Errorf("path = %q, want %q", gotPath, ApiApplication)
	}
	if name := res.Data.Name(); name != "my-app" {
		t.Errorf("name = %q, want my-app", name)
	}
}

func TestUpdate(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","name":"my-app"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), map[string]any{"name": "my-app", "contextPath": "demo"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != ApiApplication {
		t.Errorf("path = %q, want %q", gotPath, ApiApplication)
	}
	if gotBody["name"] != "my-app" || gotBody["contextPath"] != "demo" {
		t.Errorf("body = %v, want name=my-app contextPath=demo (passthrough)", gotBody)
	}
}

func TestListSubscriptions(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"users":[
			{"name":"service_user1","tenant":"t1"},
			{"name":"service_user2","tenant":"t2"}
		]}`))
	})
	defer closeFn()

	res := svc.ListSubscriptions(context.Background())
	if res.Err != nil {
		t.Fatalf("ListSubscriptions: %v", res.Err)
	}
	if gotPath != ApiApplicationSubscriptions {
		t.Errorf("path = %q, want %q", gotPath, ApiApplicationSubscriptions)
	}
	var names []string
	for u, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		names = append(names, u.Username())
	}
	if len(names) != 2 || names[0] != "service_user1" || names[1] != "service_user2" {
		t.Errorf("users = %v, want [service_user1 service_user2]", names)
	}
}
