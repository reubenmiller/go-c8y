package fakeserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
