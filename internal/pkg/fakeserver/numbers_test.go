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
		t.Fatalf("create: expected normalized number, got %s", created)
	}

	_, upd := doRequest(t, fs, http.MethodPut, "/inventory/managedObjects/"+id, `{"c8y_Kpi":{"min":0.50}}`, jsonAccept)
	if !strings.Contains(upd, `"min":0.5`) {
		t.Fatalf("update: expected normalized number, got %s", upd)
	}
	_, got := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects/"+id, "", jsonAccept)
	if !strings.Contains(got, `"min":0.5`) {
		t.Fatalf("get: expected normalized number, got %s", got)
	}
}
