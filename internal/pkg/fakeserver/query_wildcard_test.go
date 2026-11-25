package fakeserver

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestMatchWildcard(t *testing.T) {
	cases := []struct {
		pattern, actual string
		want            bool
	}{
		{"device01", "device01", true},
		{"device01", "device02", false},
		{"dev*", "device01", true},
		{"*01", "device01", true},
		{"*vice*", "device01", true},
		{"**", "device01", true},
		{"**", "", true},
		{"*", "", true},
		{"d*e*1", "device01", true},
		{"d*x*1", "device01", false},
		{"ab*ba", "aba", false},
		{"*device01*", "device01", true},
		{"agent*", "device01", false},
	}
	for _, c := range cases {
		if got := matchWildcard(c.pattern, c.actual); got != c.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", c.pattern, c.actual, got, c.want)
		}
	}
}

func TestApplyCQLFilter_Wildcards(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "1", "name": "device01", "c8y_IsDevice": map[string]any{}},
		map[string]any{"id": "2", "name": "device02", "c8y_IsDevice": map[string]any{}},
		map[string]any{"id": "3", "name": "my-device", "c8y_IsDeviceGroup": map[string]any{}},
	)
	cases := map[string][]string{
		"(name eq '**') and has(c8y_IsDevice)":                  {"1", "2"},
		"(name eq '*vice0*') and has(c8y_IsDevice)":             {"1", "2"},
		"(name eq '*02*') and has(c8y_IsDevice)":                {"2"},
		"(name eq '*device*') and has(c8y_IsDeviceGroup)":       {"3"},
		"$filter=(name eq '*-dev*') $orderby=name,creationTime": {"3"},
		"name eq 'device01'":                                    {"1"},
	}
	for query, want := range cases {
		got := ids(applyCQLFilter(items, query))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("applyCQLFilter(%q) = %v, want %v", query, got, want)
		}
	}
}

func TestFeatures_ListIsPlainArray(t *testing.T) {
	fs := New(t)
	fs.Features.Create(json.RawMessage(`{"key":"feature-a","phase":"PUBLIC_PREVIEW","active":true}`), fs.URL()+"/features")

	req, err := http.NewRequest(http.MethodGet, fs.URL()+"/features", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Fatalf("GET /features returned %s, want a JSON array", body)
	}
	var features []map[string]any
	if err := json.Unmarshal(body, &features); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range features {
		if f["key"] == "feature-a" {
			found = true
		}
	}
	if !found {
		t.Errorf("GET /features = %s, want it to contain feature-a", body)
	}
}
