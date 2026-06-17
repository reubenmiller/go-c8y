package firmwareversions

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

// TestListPlainIDSkipsFirmwareLookup verifies that a plain numeric FirmwareID is
// used directly (no extra firmware GET), the bygroupid filter is applied, and
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

	opt := ListOptions{FirmwareID: "12345", Version: "1.0.0"}
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
	if !strings.Contains(rawQuery, "c8y_Firmware.version eq '1.0.0'") {
		t.Errorf("query = %q, want version filter", rawQuery)
	}
	if withParents != "true" {
		t.Errorf("withParents = %q, want true", withParents)
	}
}

func TestIsPlainID(t *testing.T) {
	cases := map[string]bool{"": false, "12345": true, "name:linux": false, "linux": false, "12a": false}
	for in, want := range cases {
		if got := isPlainID(in); got != want {
			t.Errorf("isPlainID(%q) = %v, want %v", in, got, want)
		}
	}
}
