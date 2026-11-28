package firmwarepatches

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&httpcore.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestListPlainIDSkipsFirmwareLookup verifies that a plain numeric FirmwareID is
// used directly (no extra firmware GET), the patch scoping is applied, and
// withParents is forwarded.
func TestListPlainIDSkipsFirmwareLookup(t *testing.T) {
	var paths []string
	var rawQuery, withParents string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		rawQuery, _ = url.QueryUnescape(r.URL.RawQuery)
		withParents = r.URL.Query().Get("withParents")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	})
	defer closeFn()

	opt := ListOptions{FirmwareID: "12345", Version: "1.0.1", DependencyVersion: "1.0.0"}
	opt.WithParents = true
	if res := svc.List(context.Background(), opt); res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 request (list only, no firmware lookup), got %d: %v", len(paths), paths)
	}
	if !strings.Contains(rawQuery, "bygroupid(12345)") {
		t.Errorf("query = %q, want bygroupid(12345)", rawQuery)
	}
	if !strings.Contains(rawQuery, "has(c8y_Patch)") {
		t.Errorf("query = %q, want patch scoping has(c8y_Patch)", rawQuery)
	}
	if !strings.Contains(rawQuery, "c8y_Patch.dependency eq '1.0.0'") {
		t.Errorf("query = %q, want dependency filter", rawQuery)
	}
	if withParents != "true" {
		t.Errorf("withParents = %q, want true", withParents)
	}
}

// TestServiceResolveID verifies the service's ResolveID uses the patch
// resolver: a plain id is used as is, and a version is looked up scoped to its
// firmware.
func TestServiceResolveID(t *testing.T) {
	var rawQuery string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		rawQuery, _ = url.QueryUnescape(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[{"id":"777","c8y_Patch":{}}]}`))
	})
	defer closeFn()

	if got, err := svc.ResolveID(context.Background(), "12345", nil); err != nil || got != "12345" {
		t.Errorf("ResolveID(id) = %q, %v, want 12345", got, err)
	}
	if rawQuery != "" {
		t.Errorf("a plain id was looked up. query=%s", rawQuery)
	}

	got, err := svc.ResolveID(context.Background(), NewRef().ByVersion("1.0.1", "100"), nil)
	if err != nil || got != "777" {
		t.Errorf("ResolveID(version) = %q, %v, want 777", got, err)
	}
	if !strings.Contains(rawQuery, "bygroupid(100)") || !strings.Contains(rawQuery, "1.0.1") {
		t.Errorf("the patch was not looked up by its firmware and version. query=%s", rawQuery)
	}
}
