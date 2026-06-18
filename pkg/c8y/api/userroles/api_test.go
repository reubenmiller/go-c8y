package userroles

import (
	"context"
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

// TestRoleSelfLink verifies a bare role name is turned into its canonical self
// link from the client base URL, while a value that is already a self link
// passes through unchanged (and "" stays "").
func TestRoleSelfLink(t *testing.T) {
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {})
	defer closeFn()

	base := svc.Client.BaseURL()

	if got := svc.RoleSelfLink("ROLE_ALARM_READ"); got != base+"/user/roles/ROLE_ALARM_READ" {
		t.Errorf("RoleSelfLink(name) = %q, want %q", got, base+"/user/roles/ROLE_ALARM_READ")
	}
	existing := "https://example.com/user/roles/ROLE_ALARM_READ"
	if got := svc.RoleSelfLink(existing); got != existing {
		t.Errorf("RoleSelfLink(self) = %q, want it unchanged", got)
	}
	if got := svc.RoleSelfLink(""); got != "" {
		t.Errorf("RoleSelfLink(\"\") = %q, want empty", got)
	}
}

// TestRoleID verifies a role reference is reduced to the role id (== name) for
// the unassign path, across the forms the CLI may supply.
func TestRoleID(t *testing.T) {
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {})
	defer closeFn()

	cases := map[string]string{
		"ROLE_ALARM_READ": "ROLE_ALARM_READ",
		"https://example.com/user/roles/ROLE_ALARM_READ":    "ROLE_ALARM_READ",
		`{"id":"ROLE_ALARM_READ","name":"ROLE_ALARM_READ"}`: "ROLE_ALARM_READ",
		`{"role":{"id":"ROLE_ALARM_READ"}}`:                 "ROLE_ALARM_READ",
		"":                                                  "",
	}
	for ref, want := range cases {
		if got := svc.RoleID(ref); got != want {
			t.Errorf("RoleID(%q) = %q, want %q", ref, got, want)
		}
	}
}

// TestList verifies the role catalog is requested at /user/roles and items are
// plucked from the "roles" collection property.
func TestList(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"roles":[{"id":"ROLE_A","name":"ROLE_A"}],"statistics":{"totalPages":1}}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if !strings.HasSuffix(gotPath, "/user/roles") {
		t.Errorf("path = %q, want .../user/roles", gotPath)
	}
}
