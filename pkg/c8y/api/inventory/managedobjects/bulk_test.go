package managedobjects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

// The recovery lookup is retried, but the original bulk request never is.
func TestRecoverCreateBulk_RetriesLookupOnly(t *testing.T) {
	prev := recoverLookupBackoff
	recoverLookupBackoff = 0
	defer func() { recoverLookupBackoff = prev }()

	marker := "c8y_BatchId_abc"
	var mu sync.Mutex
	gets, posts := 0, []string{}
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			gets++
			if gets <= 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"general/serviceUnavailable","message":"busy"}`))
				return
			}
			_, _ = fmt.Fprintf(w, `{"managedObjects":[{"id":"old-0","%s":{"index":0}}]}`, marker)
		case http.MethodPost:
			posts = append(posts, string(b))
			var env struct {
				ManagedObjects []map[string]any `json:"managedObjects"`
			}
			_ = json.Unmarshal(b, &env)
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"new-1"}]}`))
		case http.MethodPut:
			_, _ = w.Write([]byte(`{"managedObjects":[]}`))
		}
	})
	defer closeFn()

	items := []any{map[string]any{"name": "item-0"}, map[string]any{"name": "item-1"}}
	out, err := svc.RecoverCreateBulk(context.Background(), items,
		&core.BatchError{Applied: core.BatchAppliedPartial, Marker: marker})
	if err != nil {
		t.Fatalf("RecoverCreateBulk: %v", err)
	}
	if len(out) != 2 || out[0].ID() != "old-0" || out[1].ID() != "new-1" {
		t.Fatalf("results = %v", out)
	}
	if gets != 3 {
		t.Errorf("lookups = %d, want 3 (two failures, one success)", gets)
	}
	if len(posts) != 1 || strings.Contains(posts[0], `"item-0"`) || !strings.Contains(posts[0], `"item-1"`) {
		t.Errorf("posts = %v, want one request with only item-1", posts)
	}
}
