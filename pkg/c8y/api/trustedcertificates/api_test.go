package trustedcertificates

import (
	"context"
	"io"
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

// TestList verifies the collection endpoint is hit and the "certificates"
// property is plucked.
func TestList(t *testing.T) {
	var gotPath, gotMethod string
	var gotQuery url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotQuery = r.URL.Path, r.Method, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"certificates":[{"fingerprint":"` + fingerprintA + `","name":"MyCert"},{"fingerprint":"deadbeef","name":"Other"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{TenantID: "t123"})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/tenant/tenants/t123/trusted-certificates") {
		t.Errorf("path = %q, want .../tenant/tenants/t123/trusted-certificates", gotPath)
	}
	if gotQuery.Has("TenantID") {
		t.Errorf("query = %q, TenantID must not leak into query parameters", gotQuery.Encode())
	}
	count := 0
	for cert, err := range res.Items() {
		if err != nil {
			t.Fatalf("item: %v", err)
		}
		_ = cert
		count++
	}
	if count != 2 {
		t.Errorf("plucked %d certificates, want 2", count)
	}
}

// TestCreate verifies the create endpoint, method and raw body passthrough.
func TestCreate(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	var gotQuery url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotQuery = r.URL.Path, r.Method, r.URL.Query()
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","name":"MyCert"}`))
	})
	defer closeFn()

	body := map[string]any{"name": "MyCert", "status": "ENABLED", "certInPemFormat": "abc"}
	res := svc.Create(context.Background(), CreateOptions{TenantID: "t123"}, body)
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/tenant/tenants/t123/trusted-certificates") {
		t.Errorf("path = %q, want .../trusted-certificates", gotPath)
	}
	if !strings.Contains(gotBody, `"certInPemFormat":"abc"`) || !strings.Contains(gotBody, `"name":"MyCert"`) {
		t.Errorf("body = %q, missing passthrough fields", gotBody)
	}
	if gotQuery.Has("TenantID") {
		t.Errorf("query = %q, TenantID must not leak into query parameters", gotQuery.Encode())
	}
}

// TestCreateWithAddToTrustStore verifies the AddToTrustStore option is sent as
// the addToTrustStore query parameter (and TenantID stays out of the query).
func TestCreateWithAddToTrustStore(t *testing.T) {
	var gotQuery url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","name":"MyCert"}`))
	})
	defer closeFn()

	addToTrustStore := false
	res := svc.Create(context.Background(), CreateOptions{
		TenantID:        "t123",
		AddToTrustStore: &addToTrustStore,
	}, map[string]any{"name": "MyCert"})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if got := gotQuery.Get("addToTrustStore"); got != "false" {
		t.Errorf("addToTrustStore query param = %q, want \"false\"", got)
	}
	if gotQuery.Has("TenantID") {
		t.Errorf("query = %q, TenantID must not leak into query parameters", gotQuery.Encode())
	}
}

// TestGet verifies the single-get endpoint uses the fingerprint path param.
func TestGet(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","name":"MyCert"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{TenantID: "t123", Fingerprint: fingerprintA})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/trusted-certificates/"+fingerprintA) {
		t.Errorf("path = %q, want .../trusted-certificates/%s", gotPath, fingerprintA)
	}
}

// TestUpdate verifies the update endpoint, method, fingerprint path and body.
func TestUpdate(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fingerprint":"` + fingerprintA + `","status":"DISABLED"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), UpdateOptions{TenantID: "t123", Fingerprint: fingerprintA}, map[string]any{"status": "DISABLED"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/trusted-certificates/"+fingerprintA) {
		t.Errorf("path = %q, want .../trusted-certificates/%s", gotPath, fingerprintA)
	}
	if !strings.Contains(gotBody, `"status":"DISABLED"`) {
		t.Errorf("body = %q, want status passthrough", gotBody)
	}
}

// TestDelete verifies the delete endpoint uses the fingerprint path param.
func TestDelete(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), DeleteOptions{TenantID: "t123", Fingerprint: fingerprintA})
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/trusted-certificates/"+fingerprintA) {
		t.Errorf("path = %q, want .../trusted-certificates/%s", gotPath, fingerprintA)
	}
}

func TestIsFingerprint(t *testing.T) {
	cases := map[string]bool{
		fingerprintA:       true,
		"MyCert":           false,
		"0123456789abcdef": false, // too short
		"0123456789ABCDEF0123456789abcdef01234567": false, // uppercase not matched (matches v1)
	}
	for in, want := range cases {
		if got := IsFingerprint(in); got != want {
			t.Errorf("IsFingerprint(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestResolveIDFingerprintPassthrough verifies a fingerprint-looking reference
// is used directly with no list lookup.
func TestResolveIDFingerprintPassthrough(t *testing.T) {
	var listCalled bool
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		listCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"certificates":[]}`))
	})
	defer closeFn()

	got, err := svc.ResolveID(context.Background(), "t123", fingerprintA)
	if err != nil {
		t.Fatalf("ResolveID: %v", err)
	}
	if got != fingerprintA {
		t.Errorf("ResolveID = %q, want %q", got, fingerprintA)
	}
	if listCalled {
		t.Error("list lookup should not be called for a fingerprint reference")
	}
}

// TestResolveIDByName verifies a name reference is resolved by listing the
// tenant's certificates and matching the name, returning the fingerprint.
func TestResolveIDByName(t *testing.T) {
	var listPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		listPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"certificates":[{"fingerprint":"deadbeef","name":"Other"},{"fingerprint":"` + fingerprintA + `","name":"MyCert"}]}`))
	})
	defer closeFn()

	got, err := svc.ResolveID(context.Background(), "t123", "MyCert")
	if err != nil {
		t.Fatalf("ResolveID: %v", err)
	}
	if got != fingerprintA {
		t.Errorf("ResolveID = %q, want %q", got, fingerprintA)
	}
	if !strings.HasSuffix(listPath, "/tenant/tenants/t123/trusted-certificates") {
		t.Errorf("list path = %q, want .../trusted-certificates", listPath)
	}
}

// TestResolveIDExplicitID verifies an "id:" reference passes through with no
// lookup.
func TestResolveIDExplicitID(t *testing.T) {
	var listCalled bool
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		listCalled = true
	})
	defer closeFn()

	got, err := svc.ResolveID(context.Background(), "t123", "id:short-id")
	if err != nil {
		t.Fatalf("ResolveID: %v", err)
	}
	if got != "short-id" {
		t.Errorf("ResolveID = %q, want short-id", got)
	}
	if listCalled {
		t.Error("list lookup should not be called for an id: reference")
	}
}
