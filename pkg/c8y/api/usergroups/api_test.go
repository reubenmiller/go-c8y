package usergroups

import (
	"context"
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

// TestGetByNameResolves verifies a name reference is resolved via the dedicated
// groupByName endpoint and the resulting id is then fetched.
func TestGetByNameResolves(t *testing.T) {
	var lookupPath, getPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/groupByName/"):
			lookupPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"42","name":"mygroup"}`))
		default:
			getPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"42","name":"mygroup"}`))
		}
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{ID: ByName("mygroup")})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if !strings.HasSuffix(lookupPath, "/groupByName/mygroup") {
		t.Errorf("lookup path = %q, want .../groupByName/mygroup", lookupPath)
	}
	if !strings.HasSuffix(getPath, "/groups/42") {
		t.Errorf("get path = %q, want .../groups/42", getPath)
	}
}

// TestGetByIDSkipsResolution verifies a plain id is used as-is, with no
// groupByName lookup.
func TestGetByIDSkipsResolution(t *testing.T) {
	var lookupCalled bool
	var getPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/groupByName/") {
			lookupCalled = true
		}
		getPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"42","name":"mygroup"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{ID: ByID("42")})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if lookupCalled {
		t.Error("groupByName lookup should not be called for a plain id")
	}
	if !strings.HasSuffix(getPath, "/groups/42") {
		t.Errorf("get path = %q, want .../groups/42", getPath)
	}
}
