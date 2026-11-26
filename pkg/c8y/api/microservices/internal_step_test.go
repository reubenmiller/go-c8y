package microservices

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

func strictCtx() context.Context {
	return ctxhelpers.WithConflictAsError(ctxhelpers.WithNotFoundAsError(context.Background()))
}

// TestDeployAlreadySubscribedWithConflictAsError verifies that redeploying an
// already subscribed microservice succeeds when the caller set
// WithConflictAsError: the 409 of the subscribe step is not an error.
func TestDeployAlreadySubscribedWithConflictAsError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "my-svc.zip")
	if err := os.WriteFile(file, []byte("PK\x03\x04 dummy zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	subscribes := 0
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/application/applications":
			_, _ = w.Write([]byte(`{"applications":[{"id":"5","name":"my-svc","type":"MICROSERVICE"}],"statistics":{"currentPage":1,"pageSize":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/application/applications/5":
			_, _ = w.Write([]byte(`{"id":"5","name":"my-svc","key":"my-svc","self":"https://x/application/applications/5"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/binaries"):
			_, _ = w.Write([]byte(`{"id":"bin1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/tenant/tenants/t100/applications":
			subscribes++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"general/conflict","message":"already subscribed"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer closeFn()

	res := svc.Deploy(strictCtx(), CreateOptions{File: file, TenantID: "t100"})
	if res.Err != nil {
		t.Fatalf("Deploy: %v", res.Err)
	}
	if res.Status != op.StatusUpdated {
		t.Errorf("status = %s, want %s", res.Status, op.StatusUpdated)
	}
	if subscribes != 1 {
		t.Errorf("subscribe requests = %d, want 1", subscribes)
	}
}

// TestSubscribeAndDeleteKeepErrorsWithOptions verifies direct calls still
// keep WithConflictAsError and WithNotFoundAsError.
func TestSubscribeAndDeleteKeepErrorsWithOptions(t *testing.T) {
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer closeFn()

	if res := svc.Subscribe(strictCtx(), "t100", "https://x/application/applications/5"); res.Err == nil {
		t.Error("Subscribe with WithConflictAsError returned no error for 409")
	}
	if res := svc.Subscribe(context.Background(), "t100", "https://x/application/applications/5"); res.Err != nil || res.Status != op.StatusDuplicate {
		t.Errorf("Subscribe without option: err=%v status=%s", res.Err, res.Status)
	}
	if res := svc.Unsubscribe(strictCtx(), "t100", "5"); res.Err == nil {
		t.Error("Unsubscribe with WithNotFoundAsError returned no error for 404")
	}
}
