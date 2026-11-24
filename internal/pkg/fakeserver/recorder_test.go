package fakeserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordingTransport_CapturesExchanges(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"123","name":"test"}`))
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(body)
		}
	}))
	defer backend.Close()

	rec := &RecordingTransport{Transport: http.DefaultTransport}
	client := &http.Client{Transport: rec}

	// GET request
	resp, err := client.Get(backend.URL + "/inventory/managedObjects/123")
	require.NoError(t, err)
	resp.Body.Close()

	// POST request with JSON body
	reqBody := []byte(`{"name":"new-object"}`)
	resp, err = client.Post(backend.URL+"/inventory/managedObjects", "application/json",
		bytes.NewReader(reqBody))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, string(body), "new-object")

	records := rec.Records()
	require.Len(t, records, 2)

	assert.Equal(t, "GET", records[0].Request.Method)
	assert.Equal(t, "/inventory/managedObjects/123", records[0].Request.Path)
	assert.Equal(t, 200, records[0].Response.StatusCode)
	assert.Contains(t, string(records[0].Response.Body), `"id":"123"`)

	assert.Equal(t, "POST", records[1].Request.Method)
	assert.Equal(t, "/inventory/managedObjects", records[1].Request.Path)
	assert.Equal(t, 201, records[1].Response.StatusCode)
	assert.Contains(t, string(records[1].Request.Body), `"name"`)
}

func TestRecordingTransport_CapturesQueryParams(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	rec := &RecordingTransport{Transport: http.DefaultTransport}
	client := &http.Client{Transport: rec}

	resp, err := client.Get(backend.URL + "/alarm/alarms?source=12345&severity=MAJOR")
	require.NoError(t, err)
	resp.Body.Close()

	records := rec.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "12345", records[0].Request.Query["source"])
	assert.Equal(t, "MAJOR", records[0].Request.Query["severity"])
}

func TestRecordingTransport_Reset(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	rec := &RecordingTransport{Transport: http.DefaultTransport}
	client := &http.Client{Transport: rec}

	resp, _ := client.Get(backend.URL + "/test")
	resp.Body.Close()
	require.Len(t, rec.Records(), 1)

	rec.Reset()
	assert.Empty(t, rec.Records())
}

func TestGoldenFile_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	rec := &RecordingTransport{Transport: http.DefaultTransport}
	rec.mu.Lock()
	rec.records = []RequestResponsePair{
		{
			Request: RecordedRequest{
				Method: "GET",
				Path:   "/alarm/alarms",
				Query:  map[string]string{"source": "123"},
			},
			Response: RecordedResponse{
				StatusCode: 200,
				Body:       json.RawMessage(`{"alarms":[]}`),
			},
		},
	}
	rec.mu.Unlock()

	require.NoError(t, rec.SaveGoldenFile(path))

	_, err := os.Stat(path)
	require.NoError(t, err)

	loaded, err := LoadGoldenFile(path)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "GET", loaded[0].Request.Method)
	assert.Equal(t, "/alarm/alarms", loaded[0].Request.Path)
	assert.Equal(t, 200, loaded[0].Response.StatusCode)
}

func TestCompareRecords_AllMatch(t *testing.T) {
	golden := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/alarm/alarms/1"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"1","type":"test","severity":"MAJOR"}`)},
		},
		{
			Request:  RecordedRequest{Method: "POST", Path: "/alarm/alarms"},
			Response: RecordedResponse{StatusCode: 201, Body: json.RawMessage(`{"id":"2","type":"test"}`)},
		},
	}

	current := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/alarm/alarms/1"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"10001","type":"fakeType","severity":"MINOR"}`)},
		},
		{
			Request:  RecordedRequest{Method: "POST", Path: "/alarm/alarms"},
			Response: RecordedResponse{StatusCode: 201, Body: json.RawMessage(`{"id":"10002","type":"fakeType"}`)},
		},
	}

	results, allPass := CompareRecords(golden, current)
	assert.True(t, allPass)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.True(t, r.StatusMatch)
		assert.True(t, r.KeysMatch)
	}
}

func TestCompareRecords_StatusMismatch(t *testing.T) {
	golden := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/alarm/alarms/1"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"1"}`)},
		},
	}
	current := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/alarm/alarms/1"},
			Response: RecordedResponse{StatusCode: 404, Body: json.RawMessage(`{"error":"not found"}`)},
		},
	}

	results, allPass := CompareRecords(golden, current)
	assert.False(t, allPass)
	require.Len(t, results, 1)
	assert.False(t, results[0].StatusMatch)
	assert.Equal(t, 200, results[0].LiveStatus)
	assert.Equal(t, 404, results[0].FakeStatus)
}

func TestCompareRecords_MissingKeys(t *testing.T) {
	golden := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/test"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"1","name":"test","extra":"field"}`)},
		},
	}
	current := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/test"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"10001","name":"fake"}`)},
		},
	}

	results, allPass := CompareRecords(golden, current)
	assert.False(t, allPass)
	require.Len(t, results, 1)
	assert.True(t, results[0].StatusMatch)
	assert.False(t, results[0].KeysMatch)
	assert.Contains(t, results[0].MissingKeys, "extra")
}

func TestCompareRecords_ExtraKeysOK(t *testing.T) {
	golden := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/test"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"1"}`)},
		},
	}
	current := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "GET", Path: "/test"},
			Response: RecordedResponse{StatusCode: 200, Body: json.RawMessage(`{"id":"10001","bonus":"field"}`)},
		},
	}

	results, allPass := CompareRecords(golden, current)
	assert.True(t, allPass)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].ExtraKeys, "bonus")
}

func TestCompareRecords_NoMatchingRequest(t *testing.T) {
	golden := []RequestResponsePair{
		{
			Request:  RecordedRequest{Method: "DELETE", Path: "/unknown/endpoint"},
			Response: RecordedResponse{StatusCode: 204},
		},
	}
	current := []RequestResponsePair{}

	results, allPass := CompareRecords(golden, current)
	assert.False(t, allPass)
	require.Len(t, results, 1)
	assert.False(t, results[0].StatusMatch)
}

func TestMergeJSON_NullRemovesField(t *testing.T) {
	got := mergeJSON(json.RawMessage(`{"id":"1","myCustom":{"value":"two"},"name":"x"}`), json.RawMessage(`{"myCustom":null,"name":"y"}`))
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["myCustom"]; ok {
		t.Errorf("myCustom should be removed: %s", got)
	}
	if m["name"] != "y" || m["id"] != "1" {
		t.Errorf("unexpected merge result: %s", got)
	}
}

func TestCertificateAuthority_CreateStoresTheCA(t *testing.T) {
	fs := New(t)
	post := func() int {
		req, _ := http.NewRequest(http.MethodPost, fs.URL()+"/certificate-authority", nil)
		req.SetBasicAuth("t12345/admin", "admin-pass")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	// The fake server is seeded with a CA certificate.
	if got := post(); got != http.StatusConflict {
		t.Fatalf("with a CA: status = %d, want 409", got)
	}
	fs.TrustedCerts.Delete("ca-fingerprint-001")
	if got := post(); got != http.StatusCreated {
		t.Fatalf("without a CA: status = %d, want 201", got)
	}
	if got := post(); got != http.StatusConflict {
		t.Fatalf("the created CA must be stored: status = %d, want 409", got)
	}
}

func TestManagedObject_WithParents(t *testing.T) {
	fs := New(t)
	do := func(method, path, body string) []byte {
		req, _ := http.NewRequest(method, fs.URL()+path, strings.NewReader(body))
		req.SetBasicAuth("t12345/admin", "admin-pass")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return b
	}
	id := func(b []byte) string {
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		return m["id"].(string)
	}
	root := id(do(http.MethodPost, "/inventory/managedObjects", `{"name":"root"}`))
	mid := id(do(http.MethodPost, "/inventory/managedObjects", `{"name":"mid"}`))
	leaf := id(do(http.MethodPost, "/inventory/managedObjects", `{"name":"leaf"}`))
	do(http.MethodPost, "/inventory/managedObjects/"+root+"/childAdditions", `{"managedObject":{"id":"`+mid+`"}}`)
	do(http.MethodPost, "/inventory/managedObjects/"+mid+"/childAdditions", `{"managedObject":{"id":"`+leaf+`"}}`)

	var got struct {
		AdditionParents struct {
			References []struct {
				ManagedObject struct{ ID, Name string } `json:"managedObject"`
			} `json:"references"`
		} `json:"additionParents"`
		AssetParents struct {
			References []any `json:"references"`
		} `json:"assetParents"`
	}
	if err := json.Unmarshal(do(http.MethodGet, "/inventory/managedObjects/"+leaf+"?withParents=true", ""), &got); err != nil {
		t.Fatal(err)
	}
	refs := got.AdditionParents.References
	if len(refs) != 2 || refs[0].ManagedObject.ID != mid || refs[1].ManagedObject.ID != root || refs[0].ManagedObject.Name != "mid" {
		t.Errorf("additionParents = %+v, want mid then root", refs)
	}
	if len(got.AssetParents.References) != 0 {
		t.Errorf("assetParents = %+v, want none", got.AssetParents.References)
	}

	plain := do(http.MethodGet, "/inventory/managedObjects/"+leaf, "")
	if strings.Contains(string(plain), "additionParents") {
		t.Errorf("parents must only be added with withParents=true: %s", plain)
	}
}
