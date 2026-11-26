package fakeserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestUsers_UsernameFilter(t *testing.T) {
	fs := New(t)
	for _, name := range []string{"alice", "Alicia", "bob"} {
		postJSON(t, fs, "/user/t12345/users", `{"userName":"`+name+`","password":"Passw0rd!x"}`)
	}
	count := func(query string) int {
		_, body := doRequest(t, fs, http.MethodGet, "/user/t12345/users?"+query, "", jsonAccept)
		var resp struct {
			Users []json.RawMessage `json:"users"`
		}
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatal(err)
		}
		return len(resp.Users)
	}
	if got := count("username=ali"); got != 2 {
		t.Errorf("username=ali (case-insensitive prefix): got %d, want 2", got)
	}
	if got := count("username=bob"); got != 1 {
		t.Errorf("username=bob: got %d, want 1", got)
	}
	if got := count("usernameContains=lic"); got != 2 {
		t.Errorf("usernameContains=lic: got %d, want 2", got)
	}
	if got := count("username=nobody"); got != 0 {
		t.Errorf("username=nobody: got %d, want 0", got)
	}
}

func TestAllTenantsSummary_IsAnArray(t *testing.T) {
	fs := New(t)
	status, body := doRequest(t, fs, http.MethodGet, "/tenant/statistics/allTenantsSummary", "", jsonAccept)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		t.Fatalf("want a JSON array, got %s: %v", body, err)
	}
	if len(items) == 0 {
		t.Error("want at least one tenant summary")
	}
}

// q is the device query: the platform adds fragmentType=c8y_IsDevice, query does not.
func TestInventory_QIsRestrictedToDevices(t *testing.T) {
	fs := New(t)
	postJSON(t, fs, "/inventory/managedObjects", `{"name":"a-device","type":"t","c8y_IsDevice":{}}`)
	postJSON(t, fs, "/inventory/managedObjects", `{"name":"a-group","type":"t"}`)

	names := func(param string) []string {
		_, body := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects?"+param+"="+url.QueryEscape("$filter=(type eq 't')"), "", jsonAccept)
		var resp struct {
			ManagedObjects []struct {
				Name string `json:"name"`
			} `json:"managedObjects"`
		}
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, mo := range resp.ManagedObjects {
			out = append(out, mo.Name)
		}
		return out
	}
	if got := names("q"); len(got) != 1 || got[0] != "a-device" {
		t.Errorf("q: got %v, want only the device", got)
	}
	if got := names("query"); len(got) != 2 {
		t.Errorf("query: got %v, want device and group", got)
	}
}

func TestMicroserviceApplication_HasRoles(t *testing.T) {
	fs := New(t)
	id := postJSON(t, fs, "/application/applications", `{"name":"ms1","key":"ms1-key","type":"MICROSERVICE","manifest":{"requiredRoles":["ROLE_ALARM_READ"]}}`)
	_, body := doRequest(t, fs, http.MethodGet, "/application/applications/"+id, "", jsonAccept)
	var app struct {
		Roles         []string `json:"roles"`
		RequiredRoles []string `json:"requiredRoles"`
	}
	if err := json.Unmarshal([]byte(body), &app); err != nil {
		t.Fatal(err)
	}
	if app.Roles == nil || len(app.RequiredRoles) != 1 {
		t.Errorf("roles=%v requiredRoles=%v in %s", app.Roles, app.RequiredRoles, body)
	}

	hosted := postJSON(t, fs, "/application/applications", `{"name":"web1","key":"web1-key","type":"HOSTED"}`)
	_, body = doRequest(t, fs, http.MethodGet, "/application/applications/"+hosted, "", jsonAccept)
	if strings.Contains(body, `"roles"`) {
		t.Errorf("only microservices get roles: %s", body)
	}
}

func TestUnknownEndpoint_IsHTML404(t *testing.T) {
	fs := New(t)
	req, _ := http.NewRequest(http.MethodGet, fs.URL()+"/no/such/endpoint", nil)
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content type %q", ct)
	}

	// A known resource keeps the JSON error.
	status, body := doRequest(t, fs, http.MethodGet, "/alarm/alarms/999999", "", jsonAccept)
	if status != http.StatusNotFound || !strings.Contains(body, `"error"`) {
		t.Errorf("known resource: status=%d body=%s", status, body)
	}
}
