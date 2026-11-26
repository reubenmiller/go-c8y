package fakeserver

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tidwall/gjson"
)

func mustStatus(t *testing.T, fs *FakeServer, method, path, body string, want int) string {
	t.Helper()
	status, out := doRequest(t, fs, method, path, body, jsonAccept)
	if status != want {
		t.Fatalf("%s %s: status=%d want %d body=%s", method, path, status, want, out)
	}
	return out
}

func jsonField(t *testing.T, doc, path string) string {
	t.Helper()
	return gjson.Get(doc, path).String()
}

func TestMicroserviceLoggers(t *testing.T) {
	fs := New(t)

	// unknown microservice
	mustStatus(t, fs, http.MethodGet, "/service/nope/loggers", "", http.StatusNotFound)

	postJSON(t, fs, "/application/applications", `{"name":"hello-microservice","type":"MICROSERVICE","key":"hello-key"}`)

	list := mustStatus(t, fs, http.MethodGet, "/service/hello-microservice/loggers", "", http.StatusOK)
	if got := jsonField(t, list, "loggers.ROOT.effectiveLevel"); got != "INFO" {
		t.Errorf("ROOT level = %q, body %s", got, list)
	}

	// a logger without a configured level inherits ROOT
	one := mustStatus(t, fs, http.MethodGet, "/service/hello-microservice/loggers/com.acme.Foo", "", http.StatusOK)
	if got := jsonField(t, one, "effectiveLevel"); got != "INFO" {
		t.Errorf("effectiveLevel = %q, body %s", got, one)
	}

	// set, and children inherit from the closest configured package
	mustStatus(t, fs, http.MethodPost, "/service/hello-microservice/loggers/com.acme", `{"configuredLevel":"debug"}`, http.StatusNoContent)
	one = mustStatus(t, fs, http.MethodGet, "/service/hello-microservice/loggers/com.acme.Foo", "", http.StatusOK)
	if got := jsonField(t, one, "effectiveLevel"); got != "DEBUG" {
		t.Errorf("inherited level = %q, body %s", got, one)
	}
	list = mustStatus(t, fs, http.MethodGet, "/service/hello-microservice/loggers", "", http.StatusOK)
	if got := jsonField(t, list, "loggers.com\\.acme.configuredLevel"); got != "DEBUG" {
		t.Errorf("configuredLevel = %q, body %s", got, list)
	}

	// null resets the logger
	mustStatus(t, fs, http.MethodPost, "/service/hello-microservice/loggers/com.acme", `{"configuredLevel":null}`, http.StatusNoContent)
	one = mustStatus(t, fs, http.MethodGet, "/service/hello-microservice/loggers/com.acme", "", http.StatusOK)
	if got := jsonField(t, one, "effectiveLevel"); got != "INFO" {
		t.Errorf("level after reset = %q, body %s", got, one)
	}

	mustStatus(t, fs, http.MethodPost, "/service/hello-microservice/loggers/com.acme", `{"configuredLevel":"LOUD"}`, http.StatusBadRequest)
}

func TestDatabrokerConnectors(t *testing.T) {
	fs := New(t)
	id := postJSON(t, fs, "/databroker/connectors", `{"name":"conn1","targetUrl":"https://example.com"}`)
	if id == "" {
		t.Fatal("no id")
	}
	got := mustStatus(t, fs, http.MethodGet, "/databroker/connectors/"+id, "", http.StatusOK)
	if jsonField(t, got, "status") != "SUSPENDED" {
		t.Errorf("default status: %s", got)
	}

	upd := mustStatus(t, fs, http.MethodPut, "/databroker/connectors/"+id, `{"status":"ACTIVE"}`, http.StatusOK)
	if jsonField(t, upd, "status") != "ACTIVE" || jsonField(t, upd, "name") != "conn1" {
		t.Errorf("update: %s", upd)
	}

	for i := 0; i < 6; i++ {
		postJSON(t, fs, "/databroker/connectors", `{"name":"more"}`)
	}
	page := mustStatus(t, fs, http.MethodGet, "/databroker/connectors?pageSize=3&currentPage=2", "", http.StatusOK)
	var parsed struct {
		Connectors []json.RawMessage `json:"connectors"`
	}
	if err := json.Unmarshal([]byte(page), &parsed); err != nil || len(parsed.Connectors) != 3 {
		t.Errorf("page 2: err=%v body=%s", err, page)
	}

	mustStatus(t, fs, http.MethodDelete, "/databroker/connectors/"+id, "", http.StatusNoContent)
	mustStatus(t, fs, http.MethodGet, "/databroker/connectors/"+id, "", http.StatusNotFound)
	mustStatus(t, fs, http.MethodPut, "/databroker/connectors/"+id, `{}`, http.StatusNotFound)
}

func TestSmartRestTemplateExternalID(t *testing.T) {
	fs := New(t)
	mustStatus(t, fs, http.MethodPost, "/inventory/templates/globalId/999/externalId", `{"type":"c8y_SmartRestDeviceIdentifier","externalId":"x-id"}`, http.StatusNotFound)

	mo := postJSON(t, fs, "/inventory/managedObjects", `{"name":"tpl"}`)
	out := mustStatus(t, fs, http.MethodPost, "/inventory/templates/globalId/"+mo+"/externalId", `{"type":"c8y_SmartRestDeviceIdentifier","externalId":"x-id"}`, http.StatusCreated)
	if jsonField(t, out, "managedObject.id") != mo {
		t.Errorf("body: %s", out)
	}
	// the binding is a normal external id
	got := mustStatus(t, fs, http.MethodGet, "/identity/externalIds/c8y_SmartRestDeviceIdentifier/x-id", "", http.StatusOK)
	if jsonField(t, got, "managedObject.id") != mo {
		t.Errorf("identity: %s", got)
	}
	mustStatus(t, fs, http.MethodPost, "/inventory/templates/globalId/"+mo+"/externalId", `{"type":"t"}`, http.StatusBadRequest)
}

func TestIdentitySearch(t *testing.T) {
	fs := New(t)
	mo := postJSON(t, fs, "/inventory/managedObjects", `{"name":"d"}`)
	mustStatus(t, fs, http.MethodPost, "/identity/globalIds/"+mo+"/externalIds", `{"type":"c8y_Serial","externalId":"s1"}`, http.StatusCreated)

	out := mustStatus(t, fs, http.MethodPost, "/identity/search", `{"externalIds":[{"type":"c8y_Serial","externalId":"s1"},{"type":"c8y_Serial","externalId":"missing"}]}`, http.StatusOK)
	var parsed struct {
		ExternalIDs []json.RawMessage `json:"externalIds"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil || len(parsed.ExternalIDs) != 1 {
		t.Fatalf("err=%v body=%s", err, out)
	}
	if getJSONPath(parsed.ExternalIDs[0], "managedObject.id") != mo {
		t.Errorf("body: %s", out)
	}
	mustStatus(t, fs, http.MethodGet, "/identity/search", "", http.StatusMethodNotAllowed)
}

func TestLogoutAndPasswordReset(t *testing.T) {
	fs := New(t)
	mustStatus(t, fs, http.MethodPost, "/user/logout", "", http.StatusNoContent)
	mustStatus(t, fs, http.MethodPost, "/user/logout/t12345/allUsers", "", http.StatusNoContent)
	mustStatus(t, fs, http.MethodGet, "/user/logout", "", http.StatusMethodNotAllowed)

	// the reset needs no credentials, and the fake server issues no reset tokens
	req, _ := http.NewRequest(http.MethodPut, fs.URL()+"/user/passwordReset", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("no body: status=%d", resp.StatusCode)
	}
	mustStatus(t, fs, http.MethodPut, "/user/passwordReset", `{"token":"abc","newPassword":"x"}`, http.StatusUnauthorized)
}

func TestUserInventoryRoleAssignments(t *testing.T) {
	fs := New(t)
	roleID := postJSON(t, fs, "/user/inventoryroles", `{"name":"restart"}`)
	other := postJSON(t, fs, "/user/inventoryroles", `{"name":"other"}`)
	base := "/user/t12345/users/admin/roles/inventory"

	created := mustStatus(t, fs, http.MethodPost, base, `{"managedObject":"123","roles":[{"id":`+roleID+`}]}`, http.StatusCreated)
	id := jsonField(t, created, "id")
	if id == "" || jsonField(t, created, "managedObject") != "123" || jsonField(t, created, "roles.0.name") != "restart" {
		t.Fatalf("created: %s", created)
	}
	var doc map[string]json.RawMessage
	_ = json.Unmarshal([]byte(created), &doc)
	if _, internal := doc["__user"]; internal {
		t.Errorf("bookkeeping field leaked: %s", created)
	}
	if string(doc["id"])[0] == '"' {
		t.Errorf("id should be a number: %s", created)
	}

	mustStatus(t, fs, http.MethodPost, base, `{"managedObject":"456","roles":[]}`, http.StatusCreated)
	mustStatus(t, fs, http.MethodPost, "/user/t12345/users/someoneelse/roles/inventory", `{"managedObject":"789","roles":[]}`, http.StatusCreated)

	var list struct {
		Items []json.RawMessage `json:"inventoryAssignments"`
	}
	out := mustStatus(t, fs, http.MethodGet, base, "", http.StatusOK)
	_ = json.Unmarshal([]byte(out), &list)
	if len(list.Items) != 2 {
		t.Errorf("assignments of admin = %d, want 2: %s", len(list.Items), out)
	}
	out = mustStatus(t, fs, http.MethodGet, base+"?managedObject=456", "", http.StatusOK)
	list.Items = nil
	_ = json.Unmarshal([]byte(out), &list)
	if len(list.Items) != 1 {
		t.Errorf("filtered assignments = %d, want 1: %s", len(list.Items), out)
	}

	got := mustStatus(t, fs, http.MethodGet, base+"/"+id, "", http.StatusOK)
	if jsonField(t, got, "managedObject") != "123" {
		t.Errorf("get: %s", got)
	}
	// another user's assignment is not reachable
	mustStatus(t, fs, http.MethodGet, "/user/t12345/users/someoneelse/roles/inventory/"+id, "", http.StatusNotFound)

	upd := mustStatus(t, fs, http.MethodPut, base+"/"+id, `{"roles":[{"id":`+other+`}]}`, http.StatusOK)
	if jsonField(t, upd, "roles.0.name") != "other" || jsonField(t, upd, "managedObject") != "123" {
		t.Errorf("update: %s", upd)
	}

	mustStatus(t, fs, http.MethodDelete, base+"/"+id, "", http.StatusNoContent)
	mustStatus(t, fs, http.MethodGet, base+"/"+id, "", http.StatusNotFound)
	mustStatus(t, fs, http.MethodDelete, base+"/"+id, "", http.StatusNotFound)
	mustStatus(t, fs, http.MethodPost, base, `{"roles":[]}`, http.StatusUnprocessableEntity)
}

func TestDevicePermissionOwners(t *testing.T) {
	fs := New(t)
	got := mustStatus(t, fs, http.MethodGet, "/user/devicePermissions/123", "", http.StatusOK)
	if jsonField(t, got, "users") != "[]" {
		t.Errorf("empty owners: %s", got)
	}
	body := `{"users":[{"userName":"admin","devicePermissions":{"123":["READ"]}}],"groups":[]}`
	mustStatus(t, fs, http.MethodPut, "/user/devicePermissions/123", body, http.StatusOK)
	got = mustStatus(t, fs, http.MethodGet, "/user/devicePermissions/123", "", http.StatusOK)
	if jsonField(t, got, "users.0.userName") != "admin" {
		t.Errorf("owners: %s", got)
	}
	// another managed object is unaffected
	got = mustStatus(t, fs, http.MethodGet, "/user/devicePermissions/456", "", http.StatusOK)
	if jsonField(t, got, "users") != "[]" {
		t.Errorf("owners of 456: %s", got)
	}
}
