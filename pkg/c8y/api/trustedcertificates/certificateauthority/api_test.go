package certificateauthority

import (
	"context"
	"io"
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

// caServer answers the CA lookup with one certificate and records the other
// requests.
type caServer struct {
	requests []string
	bodies   []string
	conflict bool
}

func (c *caServer) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := io.ReadAll(r.Body)
	switch {
	case r.Method == http.MethodGet:
		_, _ = w.Write([]byte(`{"certificates":[{"fingerprint":"` + fingerprintA + `","status":"ENABLED","autoRegistrationEnabled":true}]}`))
		return
	case r.Method == http.MethodPost && c.conflict:
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"conflict","message":"exists"}`))
	case r.Method == http.MethodPost:
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","status":"ENABLED","autoRegistrationEnabled":true}`))
	default:
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","status":"DISABLED","autoRegistrationEnabled":false}`))
	}
	c.requests = append(c.requests, r.Method+" "+r.URL.Path)
	c.bodies = append(c.bodies, string(body))
}

func TestUpdate(t *testing.T) {
	ca := &caServer{}
	svc, closeFn := testService(ca.handler)
	defer closeFn()

	res := svc.Update(context.Background(), UpdateOptions{TenantID: "t123"}, map[string]any{"status": "DISABLED"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if len(ca.requests) != 1 || ca.requests[0] != "PUT /tenant/tenants/t123/trusted-certificates/"+fingerprintA {
		t.Errorf("requests = %v", ca.requests)
	}
	if !strings.Contains(ca.bodies[0], `"status":"DISABLED"`) {
		t.Errorf("body = %s", ca.bodies[0])
	}
}

func TestCreate_AppliesSettingsThatDiffer(t *testing.T) {
	disabled := false
	for _, conflict := range []bool{false, true} {
		ca := &caServer{conflict: conflict}
		svc, closeFn := testService(ca.handler)

		res := svc.Create(context.Background(), CreateOptions{TenantID: "t123", AutoRegistrationEnabled: &disabled})
		closeFn()
		if res.Err != nil {
			t.Fatalf("Create (conflict=%v): %v", conflict, res.Err)
		}
		want := []string{"POST /certificate-authority", "PUT /tenant/tenants/t123/trusted-certificates/" + fingerprintA}
		if strings.Join(ca.requests, ",") != strings.Join(want, ",") {
			t.Errorf("conflict=%v: requests = %v, want %v", conflict, ca.requests, want)
		}
		if !strings.Contains(ca.bodies[1], `"autoRegistrationEnabled":false`) || strings.Contains(ca.bodies[1], "status") {
			t.Errorf("conflict=%v: update body = %s", conflict, ca.bodies[1])
		}
	}
}

func TestCreate_NoUpdateWhenSettingsMatch(t *testing.T) {
	enabled := true
	ca := &caServer{}
	svc, closeFn := testService(ca.handler)
	defer closeFn()

	res := svc.Create(context.Background(), CreateOptions{Status: "enabled", AutoRegistrationEnabled: &enabled})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if len(ca.requests) != 1 {
		t.Errorf("requests = %v, want only the POST", ca.requests)
	}
}
