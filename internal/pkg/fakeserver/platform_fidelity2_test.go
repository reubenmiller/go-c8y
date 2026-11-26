package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNotFound_PlatformErrors(t *testing.T) {
	fs := New(t)
	cases := []struct {
		path, errorType, message string
	}{
		{"/inventory/managedObjects/0", "inventory/Not Found", "Finding device data from database failed : No managedObject for id '0'!"},
		{"/alarm/alarms/123", "alarm/Not Found", "No alarm for id '123'"},
		{"/event/events/123", "event/Not Found", "No event for id '123'"},
		{"/measurement/measurements/123", "measurement/Not Found", "No measurement for id '123'"},
		{"/devicecontrol/operations/123", "devicecontrol/Not Found", "No operation for id '123'"},
	}
	for _, c := range cases {
		status, body := doRequest(t, fs, http.MethodGet, c.path, "", jsonAccept)
		if status != http.StatusNotFound {
			t.Errorf("%s: status=%d", c.path, status)
			continue
		}
		var e map[string]string
		if err := json.Unmarshal([]byte(body), &e); err != nil {
			t.Fatal(err)
		}
		if e["error"] != c.errorType || !strings.Contains(e["message"], c.message) || e["info"] == "" {
			t.Errorf("%s: got %v", c.path, e)
		}
	}
}

func TestMicroservices_HaveRoles(t *testing.T) {
	fs := New(t)
	_, body := doRequest(t, fs, http.MethodGet, "/application/applications?type=MICROSERVICE&pageSize=50", "", jsonAccept)
	var resp struct {
		Applications []struct {
			Name          string         `json:"name"`
			Roles         []string       `json:"roles"`
			RequiredRoles []string       `json:"requiredRoles"`
			Manifest      map[string]any `json:"manifest"`
		} `json:"applications"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Applications) == 0 {
		t.Fatal("no microservices")
	}
	for _, a := range resp.Applications {
		if len(a.Roles) == 0 || len(a.RequiredRoles) == 0 || a.Manifest == nil {
			t.Errorf("%s: roles=%v requiredRoles=%v manifest=%v", a.Name, a.Roles, a.RequiredRoles, a.Manifest)
		}
	}
}

func TestBulkRegistration_ExistingDeviceUpdatesCredentials(t *testing.T) {
	fs := New(t)
	csv := "ID,CREDENTIALS,AUTH_TYPE\nbulk-existing,secret,BASIC\n"
	status, _, out := upload(t, fs, http.MethodPost, "/devicecontrol/bulkNewDeviceRequests", "file", "r.csv", "text/csv", []byte(csv), nil)
	if status != http.StatusCreated {
		t.Fatalf("first: status=%d body=%s", status, out)
	}
	// The same password again is rejected for the row.
	status, _, out = upload(t, fs, http.MethodPost, "/devicecontrol/bulkNewDeviceRequests", "file", "r.csv", "text/csv", []byte(csv), nil)
	if status != http.StatusCreated || !strings.Contains(out, `"numberOfFailed":1`) || !strings.Contains(out, "Error during password validation.") {
		t.Fatalf("same password: status=%d body=%s", status, out)
	}
	csv = strings.Replace(csv, "secret", "secret2", 1)
	status, _, out = upload(t, fs, http.MethodPost, "/devicecontrol/bulkNewDeviceRequests", "file", "r.csv", "text/csv", []byte(csv), nil)
	if status != http.StatusOK {
		t.Fatalf("new password: status=%d body=%s", status, out)
	}
	var resp struct {
		Successful int              `json:"numberOfSuccessful"`
		Failed     int              `json:"numberOfFailed"`
		Updated    []map[string]any `json:"credentialUpdatedList"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Successful != 1 || resp.Failed != 0 || len(resp.Updated) != 1 ||
		resp.Updated[0]["deviceId"] != "bulk-existing" || resp.Updated[0]["bulkNewDeviceStatus"] != "CREDENTIAL_UPDATED" {
		t.Errorf("unexpected response %s", out)
	}
}

func TestPut_MergesFragments(t *testing.T) {
	fs := New(t)
	src := postJSON(t, fs, "/inventory/managedObjects", `{"name":"src","c8y_IsDevice":{}}`)
	for _, res := range []struct{ create, path, body string }{
		{"/inventory/managedObjects", "/inventory/managedObjects/", `{"name":"n","type":"t","keep":{"a":1},"drop":{"b":2}}`},
		{"/event/events", "/event/events/", `{"type":"t","text":"x","time":"2020-01-01T00:00:00Z","source":{"id":"SRC"},"keep":{"a":1},"drop":{"b":2}}`},
		{"/alarm/alarms", "/alarm/alarms/", `{"type":"t","text":"x","severity":"MINOR","time":"2020-01-01T00:00:00Z","source":{"id":"SRC"},"keep":{"a":1},"drop":{"b":2}}`},
		{"/devicecontrol/operations", "/devicecontrol/operations/", `{"deviceId":"SRC","keep":{"a":1},"drop":{"b":2}}`},
	} {
		id := postJSON(t, fs, res.create, strings.ReplaceAll(res.body, "SRC", src))
		status, out := doRequest(t, fs, http.MethodPut, res.path+id, `{"drop":null,"added":{"c":3}}`, jsonAccept)
		if status != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", res.path, status, out)
		}
		_, got := doRequest(t, fs, http.MethodGet, res.path+id, "", jsonAccept)
		var m map[string]any
		_ = json.Unmarshal([]byte(got), &m)
		if m["keep"] == nil || m["added"] == nil || m["drop"] != nil {
			t.Errorf("%s: PUT did not merge: %s", res.path, got)
		}
	}
}
