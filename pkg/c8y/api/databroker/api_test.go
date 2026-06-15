package databroker

import (
	"context"
	"encoding/json"
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

func TestList(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connectors":[{"id":"1","status":"ACTIVE"},{"id":"2","status":"SUSPENDED"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotPath != ApiConnectors {
		t.Errorf("path = %q, want %q", gotPath, ApiConnectors)
	}
	var ids []string
	for c, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		ids = append(ids, c.ID())
	}
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Errorf("ids = %v, want [1 2]", ids)
	}
}

func TestGetUpdate(t *testing.T) {
	var method, gotPath string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method, gotPath = r.Method, r.URL.Path
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","status":"SUSPENDED"}`))
	})
	defer closeFn()

	if res := svc.Get(context.Background(), "123"); res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotPath != "/databroker/connectors/123" {
		t.Errorf("get path = %q", gotPath)
	}

	res := svc.Update(context.Background(), "123", map[string]any{"status": "SUSPENDED"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut || gotPath != "/databroker/connectors/123" {
		t.Errorf("update = %s %s", method, gotPath)
	}
	if body["status"] != "SUSPENDED" {
		t.Errorf("update body = %v", body)
	}
}
