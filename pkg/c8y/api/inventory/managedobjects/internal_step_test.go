package managedobjects

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// TestGetOrCreateByExternalIDIdentityConflict verifies that, in the fallback
// for platforms without the externalIds field, a 409 on binding the external
// id removes the new managed object and returns the existing one, whatever
// options the caller set.
func TestGetOrCreateByExternalIDIdentityConflict(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"default": context.Background(),
		"strict":  ctxhelpers.WithConflictAsError(ctxhelpers.WithNotFoundAsError(context.Background())),
	} {
		t.Run(name, func(t *testing.T) {
			deletes := 0
			bound := false
			svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/identity/externalIds/c8y_Serial/dev1":
					if bound {
						// another client bound the id to managed object 7
						_, _ = w.Write([]byte(`{"externalId":"dev1","type":"c8y_Serial","managedObject":{"id":"7"}}`))
						return
					}
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"general/notFound","message":"nf"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/inventory/managedObjects/7":
					_, _ = w.Write([]byte(`{"id":"7","name":"existing"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/inventory/managedObjects":
					raw, _ := io.ReadAll(r.Body)
					if strings.Contains(string(raw), "externalIds") {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"error":"general/badRequest","message":"Unknown field externalIds"}`))
						return
					}
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"9","name":"dev1"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/identity/globalIds/9/externalIds":
					bound = true
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"error":"general/conflict","message":"exists"}`))
				case r.Method == http.MethodDelete:
					deletes++
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			defer closeFn()

			res := svc.GetOrCreateByExternalID(ctx, GetOrCreateByExternalIDOptions{ExternalID: "dev1", Body: map[string]any{"name": "dev1"}})
			if res.Err != nil {
				t.Fatalf("GetOrCreateByExternalID: %v", res.Err)
			}
			if res.Status != op.StatusDuplicate {
				t.Errorf("status = %q, want %q", res.Status, op.StatusDuplicate)
			}
			if res.Data.ID() != "7" {
				t.Errorf("id = %q, want the existing managed object 7", res.Data.ID())
			}
			if deletes != 1 {
				t.Errorf("orphan deletes = %d, want 1", deletes)
			}
		})
	}
}
