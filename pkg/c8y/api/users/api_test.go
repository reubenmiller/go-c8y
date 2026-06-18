package users

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

func TestRefConstructors(t *testing.T) {
	if got := ByID("admin"); got != "admin" {
		t.Errorf("ByID = %q, want admin", got)
	}
	if got := ByDeviceUser("abc123"); got != "device_abc123" {
		t.Errorf("ByDeviceUser = %q, want device_abc123", got)
	}
}

// TestList verifies the collection endpoint path, tenant path param, query
// filters and that the "users" result property is plucked.
func TestList(t *testing.T) {
	var gotPath, gotQuery string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"users":[{"id":"alice","userName":"alice"}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{
		Tenant:      "t123",
		Username:    "ali",
		Groups:      []string{"1,2"},
		OnlyDevices: true,
	})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if gotPath != "/user/t123/users" {
		t.Errorf("path = %q, want /user/t123/users", gotPath)
	}
	if !strings.Contains(gotQuery, "username=ali") {
		t.Errorf("query = %q, want username=ali", gotQuery)
	}
	if !strings.Contains(gotQuery, "onlyDevices=true") {
		t.Errorf("query = %q, want onlyDevices=true", gotQuery)
	}
	// groups passed as a single comma-separated value (matching the CLI flag)
	if !strings.Contains(gotQuery, "groups=1%2C2") && !strings.Contains(gotQuery, "groups=1,2") {
		t.Errorf("query = %q, want groups=1,2", gotQuery)
	}
	first, err := res.First()
	if err != nil {
		t.Fatalf("First: %v", err)
	}
	if first.ID() != "alice" {
		t.Errorf("id = %q, want alice", first.ID())
	}
}

func TestGet(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"alice","userName":"alice"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), GetOptions{ID: ByID("alice"), Tenant: "t123"})
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if gotPath != "/user/t123/users/alice" {
		t.Errorf("path = %q, want /user/t123/users/alice", gotPath)
	}
}

func TestGetByUsername(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"alice","userName":"alice"}`))
	})
	defer closeFn()

	res := svc.GetByUsername(context.Background(), GetByUsernameOptions{Username: "alice", Tenant: "t123"})
	if res.Err != nil {
		t.Fatalf("GetByUsername: %v", res.Err)
	}
	if gotPath != "/user/t123/userByName/alice" {
		t.Errorf("path = %q, want /user/t123/userByName/alice", gotPath)
	}
}

func TestCreatePassesBodyThrough(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"bob","userName":"bob"}`))
	})
	defer closeFn()

	// The create endpoint takes the tenant from the request context via client
	// middleware (not present on this bare test client), so the {tenantId}
	// placeholder is left unresolved here — we only assert method/body passthrough.
	body := json.RawMessage(`{"userName":"bob","email":"bob@example.com"}`)
	res := svc.Create(context.Background(), body)
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/users") {
		t.Errorf("path = %q, want .../users", gotPath)
	}
	if gotBody["userName"] != "bob" || gotBody["email"] != "bob@example.com" {
		t.Errorf("body = %v, want raw body passed through", gotBody)
	}
}

func TestUpdate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"alice","userName":"alice"}`))
	})
	defer closeFn()

	body := json.RawMessage(`{"firstName":"Alice"}`)
	res := svc.Update(context.Background(), UpdateOptions{ID: ByID("alice"), Tenant: "t123"}, body)
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/user/t123/users/alice" {
		t.Errorf("path = %q, want /user/t123/users/alice", gotPath)
	}
	if gotBody["firstName"] != "Alice" {
		t.Errorf("body = %v, want firstName=Alice", gotBody)
	}
}

func TestDelete(t *testing.T) {
	var gotMethod, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), DeleteOptions{ID: ByID("alice"), Tenant: "t123"})
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/user/t123/users/alice" {
		t.Errorf("path = %q, want /user/t123/users/alice", gotPath)
	}
}

// TestListGroupsWithUser verifies the membership endpoint path and that the
// nested "references.#.group" result property is plucked.
func TestListGroupsWithUser(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"references":[{"group":{"id":"7","name":"admins"}}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	res := svc.ListGroupsWithUser(context.Background(), ListGroupsOptions{UserID: "alice", Tenant: "t123"})
	if res.Err != nil {
		t.Fatalf("ListGroupsWithUser: %v", res.Err)
	}
	if gotPath != "/user/t123/users/alice/groups" {
		t.Errorf("path = %q, want /user/t123/users/alice/groups", gotPath)
	}
	first, err := res.First()
	if err != nil {
		t.Fatalf("First: %v", err)
	}
	if first.Get("name").String() != "admins" {
		t.Errorf("group name = %q, want admins (nested group plucked)", first.Get("name").String())
	}

	// The paginating variant should yield the same plucked group(s).
	var count int
	for g, err := range svc.ListGroupsWithUserAll(context.Background(), ListGroupsOptions{UserID: "alice", Tenant: "t123"}).Items() {
		if err != nil {
			t.Fatalf("ListGroupsWithUserAll item: %v", err)
		}
		if g.Get("name").String() != "admins" {
			t.Errorf("paginated group name = %q, want admins", g.Get("name").String())
		}
		count++
	}
	if count != 1 {
		t.Errorf("ListGroupsWithUserAll count = %d, want 1", count)
	}
}

// TestRevokeTOTPSecret verifies the admin TOTP-revoke endpoint (the gap that was
// added for the users CLI migration): DELETE on the per-user totpSecret/revoke path.
func TestRevokeTOTPSecret(t *testing.T) {
	var gotMethod, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.RevokeTOTPSecret(context.Background(), RevokeTOTPSecretOptions{ID: ByID("alice"), Tenant: "t123"})
	if res.Err != nil {
		t.Fatalf("RevokeTOTPSecret: %v", res.Err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/user/t123/users/alice/totpSecret/revoke" {
		t.Errorf("path = %q, want /user/t123/users/alice/totpSecret/revoke", gotPath)
	}
}
