package users

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

// TestList verifies the user reference collection is requested at the per-group
// users endpoint and that each item is unwrapped to its embedded user (matching
// the v1 listGroupMembership surface of a user collection, not references).
func TestList(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"references":[{"self":"http://x/user/t1/groups/42/users/peterpi%40example.com","user":{"id":"peterpi@example.com","userName":"peterpi@example.com","self":"http://x/user/t1/users/peterpi%40example.com"}}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	result := svc.List(context.Background(), ListOptions{TenantID: "t1", GroupID: "42"})
	if result.Err != nil {
		t.Fatalf("List: %v", result.Err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/users") {
		t.Errorf("path = %q, want .../user/t1/groups/42/users", gotPath)
	}
	users, err := op.ToSliceR(result)
	if err != nil {
		t.Fatalf("ToSliceR: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	if got := users[0].UserName(); got != "peterpi@example.com" {
		t.Errorf("user.UserName() = %q, want peterpi@example.com (reference unwrapped to embedded user)", got)
	}
}

// TestAssignUser verifies the user reference body is POSTed as-is to the
// per-group users endpoint and the created reference (self + nested user) is
// returned.
func TestAssignUser(t *testing.T) {
	var gotPath, gotMethod, gotBody, gotContentType string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"self":"http://x/user/t1/groups/42/users/peterpi%40example.com","user":{"id":"peterpi@example.com","userName":"peterpi@example.com"}}`))
	})
	defer closeFn()

	body := map[string]any{"user": map[string]any{"self": "http://x/user/t1/users/peterpi%40example.com"}}
	result := svc.AssignUser(context.Background(), AssignUserOptions{TenantID: "t1", GroupID: "42"}, body)
	if result.Err != nil {
		t.Fatalf("AssignUser: %v", result.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/users") {
		t.Errorf("path = %q, want .../user/t1/groups/42/users", gotPath)
	}
	if !strings.Contains(gotContentType, "userreference") {
		t.Errorf("Content-Type = %q, want a userReference media type", gotContentType)
	}
	if !strings.Contains(gotBody, `"self":"http://x/user/t1/users/peterpi%40example.com"`) {
		t.Errorf("request body = %q, want it to carry user.self", gotBody)
	}
	if got := result.Data.Self(); got != "http://x/user/t1/groups/42/users/peterpi%40example.com" {
		t.Errorf("created reference.Self() = %q, want the membership self link", got)
	}
	if got := result.Data.Name(); got != "peterpi@example.com" {
		t.Errorf("created reference.Name() = %q, want peterpi@example.com (delegates to nested user)", got)
	}
}

// TestUnassignUser verifies the user is removed at the per-group single-user
// endpoint with a DELETE.
func TestUnassignUser(t *testing.T) {
	var gotPath, gotMethod string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	result := svc.UnassignUser(context.Background(), UnassignUserOptions{TenantID: "t1", GroupID: "42", UserID: "peterpi@example.com"})
	if result.Err != nil {
		t.Fatalf("UnassignUser: %v", result.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/user/t1/groups/42/users/peterpi@example.com") {
		t.Errorf("path = %q, want .../user/t1/groups/42/users/peterpi@example.com", gotPath)
	}
}
