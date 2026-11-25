package binaries

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"resty.dev/v3"
)

func TestResolveID(t *testing.T) {
	var requests int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[
			{"id":"11","name":"firmware-1.0.bin"},
			{"id":"12","name":"firmware-2.0.bin"},
			{"id":"13","name":"config.json"}]}`))
	}))
	defer ts.Close()
	svc := NewService(&httpcore.Service{Client: resty.New().SetBaseURL(ts.URL)})
	ctx := context.Background()

	meta := map[string]any{}
	got, err := svc.ResolveID(ctx, "12345", meta)
	if err != nil || got != "12345" || meta["source"] != "direct-id" || requests != 0 {
		t.Fatalf("id: got=%q err=%v meta=%v requests=%d", got, err, meta, requests)
	}

	meta = map[string]any{}
	got, err = svc.ResolveID(ctx, "config.json", meta)
	if err != nil || got != "13" || meta["name"] != "config.json" {
		t.Fatalf("name: got=%q err=%v meta=%v", got, err, meta)
	}

	got, err = svc.ResolveID(ctx, "*2.0*", nil)
	if err != nil || got != "12" {
		t.Fatalf("wildcard: got=%q err=%v", got, err)
	}

	_, err = svc.ResolveID(ctx, "firmware-*", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected an ambiguity error. err=%v", err)
	}

	_, err = svc.ResolveID(ctx, "missing.bin", nil)
	if !core.IsNotFound(err) {
		t.Errorf("expected not found. err=%v", err)
	}
}
