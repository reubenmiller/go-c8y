package certificateauthority

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

const fingerprintA = "0123456789abcdef0123456789abcdef01234567"

// TestGet verifies the CA lookup filters by certificateAuthority and does not
// leak the TenantID option into the query parameters.
func TestGet(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"certificates":[{"fingerprint":"` + fingerprintA + `","name":"CA Certificate"}]}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{TenantID: "t123"})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if !strings.HasSuffix(gotPath, "/tenant/tenants/t123/trusted-certificates") {
		t.Errorf("path = %q, want .../tenant/tenants/t123/trusted-certificates", gotPath)
	}
	if got := gotQuery.Get("certificateAuthority"); got != "true" {
		t.Errorf("certificateAuthority query param = %q, want \"true\"", got)
	}
	if gotQuery.Has("TenantID") {
		t.Errorf("query = %q, TenantID must not leak into query parameters", gotQuery.Encode())
	}
	if res.Data.Fingerprint() != fingerprintA {
		t.Errorf("fingerprint = %q, want %q", res.Data.Fingerprint(), fingerprintA)
	}
}
