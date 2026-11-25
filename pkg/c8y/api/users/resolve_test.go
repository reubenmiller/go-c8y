package users

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

func TestResolveID(t *testing.T) {
	var requests int
	var query string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		requests++
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"users":[{"id":"admin","userName":"admin"},{"id":"administrator","userName":"administrator"},{"id":"bob","userName":"bob"}]}`))
	})
	defer closeFn()
	ctx := context.Background()

	// exact: used as-is, no request
	meta := map[string]any{}
	got, err := svc.ResolveID(ctx, "admin", meta)
	if err != nil || got != "admin" || meta["source"] != "direct-id" || requests != 0 {
		t.Fatalf("exact: got=%q err=%v meta=%v requests=%d", got, err, meta, requests)
	}

	// wildcard, single match
	meta = map[string]any{}
	got, err = svc.ResolveID(ctx, "bo*", meta)
	if err != nil || got != "bob" || meta["source"] != "name" || meta["name"] != "bob" {
		t.Fatalf("wildcard: got=%q err=%v meta=%v", got, err, meta)
	}
	if !strings.Contains(query, "username=bo") {
		t.Errorf("expected the username prefix filter. query=%q", query)
	}

	// several matches
	got, err = svc.ResolveID(ctx, "admin*", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected an ambiguity error. got=%q err=%v", got, err)
	}

	// not found
	_, err = svc.ResolveID(ctx, "zz*", nil)
	if !core.IsNotFound(err) {
		t.Errorf("expected not found. err=%v", err)
	}

	if _, err := svc.ResolveID(ctx, " ", nil); err == nil {
		t.Error("expected an error for an empty identifier")
	}
}
