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

// TestGetOrCreateByExternalIDIdentityStepIgnoresOptions verifies the identity
// step of the fallback for platforms without the externalIds field is not
// affected by WithConflictAsError: the result matches the one without it.
func TestGetOrCreateByExternalIDIdentityStepIgnoresOptions(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"default": context.Background(),
		"strict":  ctxhelpers.WithConflictAsError(ctxhelpers.WithNotFoundAsError(context.Background())),
	} {
		t.Run(name, func(t *testing.T) {
			deletes := 0
			svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/identity/externalIds/c8y_Serial/dev1":
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"general/notFound","message":"nf"}`))
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

			first := svc.GetOrCreateByExternalID(ctx, GetOrCreateByExternalIDOptions{ExternalID: "dev1", Body: map[string]any{"name": "dev1"}})
			second := svc.GetOrCreateByExternalID(context.Background(), GetOrCreateByExternalIDOptions{ExternalID: "dev1", Body: map[string]any{"name": "dev1"}})
			if (first.Err == nil) != (second.Err == nil) || first.Status != second.Status {
				t.Errorf("with options: err=%v status=%s; without: err=%v status=%s", first.Err, first.Status, second.Err, second.Status)
			}
			if deletes != 0 {
				t.Errorf("managed object was deleted %d times", deletes)
			}
			if first.Status != op.StatusCreated {
				t.Errorf("status = %q, want %q", first.Status, op.StatusCreated)
			}
		})
	}
}
