package fakeserver

import (
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeJSONNumbers(t *testing.T) {
	cases := map[string]string{
		`{"max":19.1010101E19}`:     `{"max":1.91010101E20}`,
		`{"a": 1.5, "b": 2.50}`:     `{"a": 1.5, "b": 2.5}`,
		`{"a": 1e3}`:                `{"a": 1000.0}`,
		`{"a": 1.0}`:                `{"a": 1.0}`,
		`{"a": 12345678.5}`:         `{"a": 1.23456785E7}`,
		`{"a": 0.0001}`:             `{"a": 1.0E-4}`,
		`{"a": -2.5e-5}`:            `{"a": -2.5E-5}`,
		`{"a": 42, "b": -7}`:        `{"a": 42, "b": -7}`,
		`{"s": "1.5E3 \" 2.50"}`:    `{"s": "1.5E3 \" 2.50"}`,
		`{"l": [1.10, {"x": 3E2}]}`: `{"l": [1.1, {"x": 300.0}]}`,
		`{"id": 9007199254740993}`:  `{"id": 9007199254740993}`,
	}
	for in, want := range cases {
		if got := string(normalizeJSONNumbers([]byte(in))); got != want {
			t.Errorf("normalizeJSONNumbers(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestManagedObjectExponentNumber(t *testing.T) {
	fs := New(t)
	id := postJSON(t, fs, "/inventory/managedObjects", `{"c8y_Kpi":{"max":19.1010101E19}}`)
	_, created := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects/"+id, "", jsonAccept)
	if !strings.Contains(created, `"max":1.91010101E20`) {
		t.Fatalf("create: expected normalised number, got %s", created)
	}

	_, upd := doRequest(t, fs, http.MethodPut, "/inventory/managedObjects/"+id, `{"c8y_Kpi":{"min":0.50}}`, jsonAccept)
	if !strings.Contains(upd, `"min":0.5`) {
		t.Fatalf("update: expected normalised number, got %s", upd)
	}
	_, got := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects/"+id, "", jsonAccept)
	if !strings.Contains(got, `"min":0.5`) {
		t.Fatalf("get: expected normalised number, got %s", got)
	}
}

// A non-JSON request body (here a raw application binary upload) must be
// stored byte for byte: the number normalisation only applies to JSON.
func TestRawBinaryUploadIsNotNormalized(t *testing.T) {
	fs := New(t)
	app := postJSON(t, fs, "/application/applications", `{"name":"raw1","key":"raw1-key","contextPath":"raw1","type":"HOSTED"}`)
	content := `version 2.50 size 1e3 "x" 19.1010101E19`
	status, meta := doRequest(t, fs, http.MethodPost, "/application/applications/"+app+"/binaries", content, jsonAccept)
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("upload: status=%d body=%s", status, meta)
	}
	id := getJSONString([]byte(meta), "id")
	_, got := doRequest(t, fs, http.MethodGet, "/application/applications/"+app+"/binaries/"+id, "", "")
	if got != content {
		t.Fatalf("binary changed: got %q, want %q", got, content)
	}
}

// An application binary that is itself a JSON file is stored unchanged too.
func TestRawJSONBinaryUploadIsNotNormalized(t *testing.T) {
	fs := New(t)
	app := postJSON(t, fs, "/application/applications", `{"name":"raw2","key":"raw2-key","contextPath":"raw2","type":"HOSTED"}`)
	content := `{"version": 2.50, "size": 1e3}`
	_, meta := doRequest(t, fs, http.MethodPost, "/application/applications/"+app+"/binaries", content, jsonAccept)
	id := getJSONString([]byte(meta), "id")
	_, got := doRequest(t, fs, http.MethodGet, "/application/applications/"+app+"/binaries/"+id, "", "")
	if got != content {
		t.Fatalf("binary changed: got %q, want %q", got, content)
	}
}
