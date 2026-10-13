package currentmicroservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func TestListSettingsFlatMap(t *testing.T) {
	var path string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key1":"v1","a.b":"v2"}`))
	}))
	defer ts.Close()
	svc := NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)})

	res := svc.ListSettings(context.Background())
	if res.Err != nil {
		t.Fatalf("ListSettings: %v", res.Err)
	}
	if path != "/application/currentApplication/settings" {
		t.Errorf("path = %q", path)
	}
	m := res.Data.Map()
	if len(m) != 2 || m["key1"] != "v1" || m["a.b"] != "v2" {
		t.Errorf("map = %v", m)
	}
	if v, ok := res.Data.Value("a.b"); !ok || v != "v2" {
		t.Errorf("Value(a.b) = %q, %v", v, ok)
	}
	if _, ok := res.Data.Value("missing"); ok {
		t.Error("missing key reported present")
	}
}
