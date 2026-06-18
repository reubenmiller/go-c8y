package registration

import (
	"context"
	"io"
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

// TestList verifies the collection endpoint is hit and the "newDeviceRequests"
// property is plucked.
func TestList(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"newDeviceRequests":[{"id":"dev-1","status":"WAITING_FOR_CONNECTION"},{"id":"dev-2","status":"PENDING_ACCEPTANCE"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests") {
		t.Errorf("path = %q, want .../devicecontrol/newDeviceRequests", gotPath)
	}
	count := 0
	for item, err := range res.Items() {
		if err != nil {
			t.Fatalf("item: %v", err)
		}
		_ = item
		count++
	}
	if count != 2 {
		t.Errorf("plucked %d device requests, want 2", count)
	}
}

// TestListAll verifies the paginating iterator plucks items across the
// collection endpoint.
func TestListAll(t *testing.T) {
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"newDeviceRequests":[{"id":"dev-1"}],"statistics":{"totalPages":1,"currentPage":1}}`))
	})
	defer closeFn()

	count := 0
	for item, err := range svc.ListAll(context.Background(), ListOptions{}).Items() {
		if err != nil {
			t.Fatalf("item: %v", err)
		}
		_ = item
		count++
	}
	if count != 1 {
		t.Errorf("iterated %d device requests, want 1", count)
	}
}

// TestGet verifies the single-get endpoint uses the id path param.
func TestGet(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dev-1"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "dev-1")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests/dev-1") {
		t.Errorf("path = %q, want .../newDeviceRequests/dev-1", gotPath)
	}
}

// TestCreate verifies the create endpoint, method and typed-option body.
func TestCreate(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"dev-1"}`))
	})
	defer closeFn()

	res := svc.Create(context.Background(), CreateOptions{ID: "dev-1", Type: "c8y_Linux", GroupID: "123"})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests") {
		t.Errorf("path = %q, want .../newDeviceRequests", gotPath)
	}
	if !strings.Contains(gotBody, `"id":"dev-1"`) || !strings.Contains(gotBody, `"groupId":"123"`) {
		t.Errorf("body = %q, missing typed fields", gotBody)
	}
}

// TestCreateRaw verifies the raw body passes through unchanged to the create
// endpoint.
func TestCreateRaw(t *testing.T) {
	var gotPath, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"dev-1"}`))
	})
	defer closeFn()

	res := svc.CreateRaw(context.Background(), map[string]any{"id": "dev-1", "extra": "kept"})
	if res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests") {
		t.Errorf("path = %q, want .../newDeviceRequests", gotPath)
	}
	if !strings.Contains(gotBody, `"extra":"kept"`) {
		t.Errorf("body = %q, want raw passthrough", gotBody)
	}
}

// TestUpdate verifies the update endpoint, method, id path and typed body.
func TestUpdate(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dev-1","status":"ACCEPTED"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), "dev-1", UpdateOptions{Status: "ACCEPTED"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests/dev-1") {
		t.Errorf("path = %q, want .../newDeviceRequests/dev-1", gotPath)
	}
	if !strings.Contains(gotBody, `"status":"ACCEPTED"`) {
		t.Errorf("body = %q, want status passthrough", gotBody)
	}
}

// TestUpdateRaw verifies the raw body passes through to the update endpoint.
func TestUpdateRaw(t *testing.T) {
	var gotMethod, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dev-1"}`))
	})
	defer closeFn()

	res := svc.UpdateRaw(context.Background(), "dev-1", map[string]any{"status": "ACCEPTED", "securityToken": "tok"})
	if res.Err != nil {
		t.Fatalf("UpdateRaw: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.Contains(gotBody, `"securityToken":"tok"`) {
		t.Errorf("body = %q, want raw passthrough", gotBody)
	}
}

// TestDelete verifies the delete endpoint uses the id path param.
func TestDelete(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), "dev-1")
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/newDeviceRequests/dev-1") {
		t.Errorf("path = %q, want .../newDeviceRequests/dev-1", gotPath)
	}
}

// TestCreateCredentials verifies credentials are requested from the
// device-credentials endpoint (NOT the new-device-requests endpoint) — a
// regression guard for the previously-shared create builder.
func TestCreateCredentials(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"dev-1","username":"device_dev-1","password":"secret"}`))
	})
	defer closeFn()

	res := svc.CreateCredentials(context.Background(), CreateCredentialsOptions{ID: "dev-1"})
	if res.Err != nil {
		t.Fatalf("CreateCredentials: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/deviceCredentials") {
		t.Errorf("path = %q, want .../devicecontrol/deviceCredentials", gotPath)
	}
	if !strings.Contains(gotContentType, "devicecredentials") {
		t.Errorf("content-type = %q, want device credentials mime type", gotContentType)
	}
}

// TestCreateCredentialsRaw verifies the raw body passes through to the
// device-credentials endpoint.
func TestCreateCredentialsRaw(t *testing.T) {
	var gotPath, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"dev-1"}`))
	})
	defer closeFn()

	res := svc.CreateCredentialsRaw(context.Background(), map[string]any{"id": "dev-1"})
	if res.Err != nil {
		t.Fatalf("CreateCredentialsRaw: %v", res.Err)
	}
	if !strings.HasSuffix(gotPath, "/devicecontrol/deviceCredentials") {
		t.Errorf("path = %q, want .../devicecontrol/deviceCredentials", gotPath)
	}
	if !strings.Contains(gotBody, `"id":"dev-1"`) {
		t.Errorf("body = %q, want raw passthrough", gotBody)
	}
}
