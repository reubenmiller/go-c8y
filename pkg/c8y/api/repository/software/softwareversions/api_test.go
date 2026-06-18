package softwareversions

import (
	"context"
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

// TestListPlainIDSkipsSoftwareLookup verifies that a plain numeric SoftwareID is
// used directly (no extra software GET), the bygroupid filter is applied, the
// type/version scoping is added, and the GetOptions detail flags are forwarded.
func TestListPlainIDSkipsSoftwareLookup(t *testing.T) {
	var paths []string
	var rawQuery, withParents, withChildren string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		rawQuery, _ = url.QueryUnescape(r.URL.RawQuery)
		withParents = r.URL.Query().Get("withParents")
		withChildren = r.URL.Query().Get("withChildren")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	})
	defer closeFn()

	opt := ListOptions{SoftwareID: "12345", Version: "1.0.0"}
	opt.WithParents = true
	opt.WithChildren = true
	if res := svc.List(context.Background(), opt); res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 request (list only, no software lookup), got %d: %v", len(paths), paths)
	}
	for _, want := range []string{"type eq 'c8y_SoftwareBinary'", "c8y_Software.version eq '1.0.0'", "bygroupid(12345)"} {
		if !strings.Contains(rawQuery, want) {
			t.Errorf("query = %q, want it to contain %q", rawQuery, want)
		}
	}
	if withParents != "true" {
		t.Errorf("withParents = %q, want true", withParents)
	}
	if withChildren != "true" {
		t.Errorf("withChildren = %q, want true", withChildren)
	}
}

// TestListEmptySoftwareListsAll verifies that an empty SoftwareID lists all
// versions without a bygroupid filter (and without a software lookup), matching
// the v1 "list all software versions" behaviour.
func TestListEmptySoftwareListsAll(t *testing.T) {
	var paths []string
	var rawQuery string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		rawQuery, _ = url.QueryUnescape(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	})
	defer closeFn()

	if res := svc.List(context.Background(), ListOptions{}); res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 request (list only, no software lookup), got %d: %v", len(paths), paths)
	}
	if !strings.Contains(rawQuery, "type eq 'c8y_SoftwareBinary'") {
		t.Errorf("query = %q, want type scoping", rawQuery)
	}
	if strings.Contains(rawQuery, "bygroupid") {
		t.Errorf("query = %q, want no bygroupid for empty software", rawQuery)
	}
}

func TestIsPlainID(t *testing.T) {
	cases := map[string]bool{"": false, "12345": true, "name:python3": false, "python3": false, "12a": false}
	for in, want := range cases {
		if got := isPlainID(in); got != want {
			t.Errorf("isPlainID(%q) = %v, want %v", in, got, want)
		}
	}
}
