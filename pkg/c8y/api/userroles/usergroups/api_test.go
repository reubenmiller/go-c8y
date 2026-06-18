package usergroups

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestList verifies the role reference collection is requested at the
// per-group roles endpoint and that each item is the full reference (self +
// nested role).
func TestList(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"references":[{"self":"http://x/user/t1/groups/42/roles/ROLE_A","role":{"id":"ROLE_A","name":"ROLE_A","self":"http://x/user/roles/ROLE_A"}}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	result := svc.List(context.Background(), ListOptions{TenantID: "t1", GroupID: "42"})
	if result.Err != nil {
		t.Fatalf("List: %v", result.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/roles") {
		t.Errorf("path = %q, want .../user/t1/groups/42/roles", gotPath)
	}
	refs, err := op.ToSliceR(result)
	if err != nil {
		t.Fatalf("ToSliceR: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d references, want 1", len(refs))
	}
	if got := refs[0].Name(); got != "ROLE_A" {
		t.Errorf("reference.Name() = %q, want ROLE_A (delegates to nested role)", got)
	}
}

// TestAssignRole verifies the role reference body is POSTed as-is to the
// per-group roles endpoint and the created reference is returned.
func TestAssignRole(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"self":"http://x/user/t1/groups/42/roles/ROLE_A","role":{"id":"ROLE_A","name":"ROLE_A"}}`))
	})
	defer closeFn()

	body := map[string]any{"role": map[string]any{"self": "http://x/user/roles/ROLE_A"}}
	result := svc.AssignRole(context.Background(), AssignRoleOptions{TenantID: "t1", GroupID: "42"}, body)
	if result.Err != nil {
		t.Fatalf("AssignRole: %v", result.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/roles") {
		t.Errorf("path = %q, want .../user/t1/groups/42/roles", gotPath)
	}
	if !strings.Contains(gotBody, `"self":"http://x/user/roles/ROLE_A"`) {
		t.Errorf("request body = %q, want it to carry role.self", gotBody)
	}
	if got := result.Data.Name(); got != "ROLE_A" {
		t.Errorf("created reference.Name() = %q, want ROLE_A", got)
	}
}

// TestUnassignRole verifies the role is removed at the per-group single-role
// endpoint with a DELETE.
func TestUnassignRole(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	result := svc.UnassignRole(context.Background(), UnassignRoleOptions{TenantID: "t1", GroupID: "42", RoleID: "ROLE_A"})
	if result.Err != nil {
		t.Fatalf("UnassignRole: %v", result.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/roles/ROLE_A") {
		t.Errorf("path = %q, want .../user/t1/groups/42/roles/ROLE_A", gotPath)
	}
}
