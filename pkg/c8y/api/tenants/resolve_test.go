package tenants

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

func TestResolveID(t *testing.T) {
	var requests int
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tenants":[
			{"id":"t100","domain":"acme.example.com","company":"Acme"},
			{"id":"t200","domain":"acme-eu.example.com","company":"Acme Europe"},
			{"id":"t300","domain":"other.example.com","company":"Other Ltd"}]}`))
	})
	defer closeFn()
	ctx := context.Background()

	meta := map[string]any{}
	got, err := svc.ResolveID(ctx, "t12345", meta)
	if err != nil || got != "t12345" || meta["source"] != "direct-id" || requests != 0 {
		t.Fatalf("id: got=%q err=%v meta=%v requests=%d", got, err, meta, requests)
	}

	// by domain, exact
	meta = map[string]any{}
	got, err = svc.ResolveID(ctx, "other.example.com", meta)
	if err != nil || got != "t300" || meta["company"] != "Other Ltd" {
		t.Fatalf("domain: got=%q err=%v meta=%v", got, err, meta)
	}

	// by company, exact wins over the wildcard-capable other "Acme Europe"
	got, err = svc.ResolveID(ctx, "Acme", nil)
	if err != nil || got != "t100" {
		t.Fatalf("company: got=%q err=%v", got, err)
	}

	// wildcard
	got, err = svc.ResolveID(ctx, "oth*", nil)
	if err != nil || got != "t300" {
		t.Fatalf("wildcard: got=%q err=%v", got, err)
	}

	_, err = svc.ResolveID(ctx, "acme*", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected an ambiguity error. err=%v", err)
	}

	_, err = svc.ResolveID(ctx, "nope.example.com", nil)
	if !core.IsNotFound(err) {
		t.Errorf("expected not found. err=%v", err)
	}
}
