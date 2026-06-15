package applications

import (
	"context"
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

// TestListBinaries verifies the attachments collection is plucked into items.
func TestListBinaries(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"attachments":[{"id":"b1","name":"app.zip"},{"id":"b2","name":"app2.zip"}]}`))
	})
	defer closeFn()

	res := svc.ListBinaries(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("ListBinaries: %v", res.Err)
	}
	if path != "/application/applications/123/binaries" {
		t.Errorf("path = %q, want .../123/binaries", path)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "b1" || ids[1] != "b2" {
		t.Errorf("ids = %v, want [b1 b2]", ids)
	}
}

// TestDeleteBinary verifies the binary id is sent on the delete path.
func TestDeleteBinary(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.DeleteBinary(context.Background(), "123", "b1")
	if res.Err != nil {
		t.Fatalf("DeleteBinary: %v", res.Err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", method)
	}
	if path != "/application/applications/123/binaries/b1" {
		t.Errorf("path = %q, want .../123/binaries/b1", path)
	}
}

// TestListBinariesResolvesName verifies a name reference is resolved (via the
// applications listing) before the binaries are fetched.
func TestListBinariesResolvesName(t *testing.T) {
	var binariesPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/application/applications":
			// name lookup
			_, _ = w.Write([]byte(`{"applications":[{"id":"42","name":"cockpit"}]}`))
		default:
			binariesPath = r.URL.Path
			_, _ = w.Write([]byte(`{"attachments":[]}`))
		}
	})
	defer closeFn()

	if res := svc.ListBinaries(context.Background(), "name:cockpit"); res.Err != nil {
		t.Fatalf("ListBinaries: %v", res.Err)
	}
	if binariesPath != "/application/applications/42/binaries" {
		t.Errorf("binaries path = %q, want .../42/binaries", binariesPath)
	}
}
