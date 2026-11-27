package firmwarepatches

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"resty.dev/v3"
)

// TestGetSendsAllGetOptions verifies that every managed object get option
// reaches the request of Get (withGroups, withChildren, ... were dropped).
func TestGetSendsAllGetOptions(t *testing.T) {
	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"12","name":"x"}`))
	}))
	defer ts.Close()
	svc := NewService(&httpcore.Service{Client: resty.New().SetBaseURL(ts.URL)})

	opt := GetOptions{
		WithParents:       true,
		WithChildren:      true,
		WithChildrenCount: true,
		SkipChildrenNames: true,
		WithGroups:        true,
	}
	if res := svc.Get(context.Background(), "12", opt); res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotPath != "/inventory/managedObjects/12" {
		t.Errorf("path = %q", gotPath)
	}
	want := "skipChildrenNames=true&withChildren=true&withChildrenCount=true&withGroups=true&withParents=true"
	if gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}
