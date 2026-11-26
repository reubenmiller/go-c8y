package applications

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

func strictCtx() context.Context {
	return ctxhelpers.WithConflictAsError(ctxhelpers.WithNotFoundAsError(context.Background()))
}

// TestDeployHostedAlreadyActiveWithConflictAsError verifies that the 409 of
// activating an already active version is not an error when the caller set
// WithConflictAsError: the option applies to the deploy, not to its steps.
func TestDeployHostedAlreadyActiveWithConflictAsError(t *testing.T) {
	zip := filepath.Join(t.TempDir(), "myapp.zip")
	if err := os.WriteFile(zip, []byte("PK\x03\x04 dummy zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var deleted []string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/application/applications":
			_, _ = w.Write([]byte(`{"applications":[{"id":"5","name":"myapp","type":"HOSTED"}],"statistics":{"currentPage":1,"pageSize":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/application/applications/5":
			_, _ = w.Write([]byte(`{"id":"5","name":"myapp","type":"HOSTED","activeVersionId":"bin2"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/application/applications/5/binaries":
			_, _ = w.Write([]byte(`{"id":"bin2"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/application/applications/5":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"general/conflict","message":"already active"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/application/applications/5/binaries":
			_, _ = w.Write([]byte(`{"attachments":[{"id":"bin0","created":"2026-01-01T00:00:00.000Z"},{"id":"bin2","created":"2026-01-02T00:00:00.000Z"}]}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/application/applications/5/binaries/"):
			// the old binary is already gone
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/application/applications/5/binaries/"))
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"general/notFound","message":"gone"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer closeFn()

	res := svc.DeployHosted(strictCtx(), DeployHostedOptions{Path: zip, KeepBinaries: 1})
	if res.Err != nil {
		t.Fatalf("DeployHosted: %v", res.Err)
	}
	if res.Status != op.StatusUpdated {
		t.Errorf("status = %s, want %s", res.Status, op.StatusUpdated)
	}
	if res.Data.ID() != "5" {
		t.Errorf("id = %q, want 5", res.Data.ID())
	}
	if !slices.Equal(deleted, []string{"bin0"}) {
		t.Errorf("deleted binaries = %v, want [bin0]", deleted)
	}
}

// TestRotateBinariesMissingBinaryWithNotFoundAsError verifies the cleanup
// step of a deploy ignores a binary which is already gone.
func TestRotateBinariesMissingBinaryWithNotFoundAsError(t *testing.T) {
	b := newBinaryServer(3, "b3")
	b.missing = map[string]bool{"b1": true}
	svc, closeFn := testService(b.handler)
	defer closeFn()
	res := svc.rotateBinaries(strictCtx(), "123", RotateBinariesOptions{Keep: 1})
	if res.Err != nil {
		t.Fatalf("rotateBinaries: %v", res.Err)
	}
}

// TestDeleteBinaryKeepsNotFoundWithOption verifies a direct DeleteBinary still
// honors WithNotFoundAsError.
func TestDeleteBinaryKeepsNotFoundWithOption(t *testing.T) {
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer closeFn()
	if res := svc.DeleteBinary(strictCtx(), "123", "b1"); res.Err == nil {
		t.Error("DeleteBinary with WithNotFoundAsError returned no error for 404")
	}
	if res := svc.DeleteBinary(context.Background(), "123", "b1"); res.Err != nil {
		t.Errorf("DeleteBinary without option: %v", res.Err)
	}
}
