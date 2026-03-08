package fakeserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// RecordingTransport wraps an http.RoundTripper and captures request/response
// pairs for golden-file validation. Used in "record" mode to record live server
// exchanges that can later be compared against fake server output.
type RecordingTransport struct {
	Transport http.RoundTripper
	mu        sync.Mutex
	records   []RequestResponsePair
}

// RequestResponsePair holds a single captured HTTP exchange.
type RequestResponsePair struct {
	Request  RecordedRequest  `json:"request"`
	Response RecordedResponse `json:"response"`
}

// RecordedRequest captures the essential parts of an HTTP request.
type RecordedRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
	Body   json.RawMessage   `json:"body,omitempty"`
}

// RecordedResponse captures the essential parts of an HTTP response.
type RecordedResponse struct {
	StatusCode int             `json:"statusCode"`
	Body       json.RawMessage `json:"body,omitempty"`
}

// RoundTrip implements http.RoundTripper. It delegates to the wrapped transport
// and records the exchange.
func (rt *RecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Capture the request body before it's consumed
	var reqBody json.RawMessage
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if json.Valid(bodyBytes) && len(bodyBytes) > 0 {
			reqBody = bodyBytes
		}
	}

	// Build query map
	var queryMap map[string]string
	if len(req.URL.Query()) > 0 {
		queryMap = make(map[string]string, len(req.URL.Query()))
		for k, v := range req.URL.Query() {
			queryMap[k] = strings.Join(v, ",")
		}
	}

	// Perform the actual request
	resp, err := rt.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Capture the response body (re-buffer it for the caller)
	var respBody json.RawMessage
	if resp.Body != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr == nil {
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			if json.Valid(bodyBytes) && len(bodyBytes) > 0 {
				respBody = bodyBytes
			}
		}
	}

	pair := RequestResponsePair{
		Request: RecordedRequest{
			Method: req.Method,
			Path:   req.URL.Path,
			Query:  queryMap,
			Body:   reqBody,
		},
		Response: RecordedResponse{
			StatusCode: resp.StatusCode,
			Body:       respBody,
		},
	}

	rt.mu.Lock()
	rt.records = append(rt.records, pair)
	rt.mu.Unlock()

	return resp, nil
}

// Records returns a copy of all recorded request/response pairs.
func (rt *RecordingTransport) Records() []RequestResponsePair {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]RequestResponsePair, len(rt.records))
	copy(out, rt.records)
	return out
}

// Reset clears all recorded pairs.
func (rt *RecordingTransport) Reset() {
	rt.mu.Lock()
	rt.records = nil
	rt.mu.Unlock()
}

// SaveGoldenFile writes all recorded pairs to a JSON file at the given path.
// Parent directories are created automatically.
func (rt *RecordingTransport) SaveGoldenFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rt.Records(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadGoldenFile reads a golden file and returns the recorded pairs.
func LoadGoldenFile(path string) ([]RequestResponsePair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pairs []RequestResponsePair
	if err := json.Unmarshal(data, &pairs); err != nil {
		return nil, err
	}
	return pairs, nil
}

// GoldenFilePath returns the standard golden file location for a given test name.
func GoldenFilePath(testName string) string {
	return filepath.Join("testdata", "golden", testName+".json")
}

// CompareResult holds the outcome of comparing a recorded pair against an
// offline pair for the same request.
type CompareResult struct {
	Request     RecordedRequest `json:"request"`
	StatusMatch bool            `json:"statusMatch"`
	KeysMatch   bool            `json:"keysMatch"`
	LiveStatus  int             `json:"liveStatus"`
	FakeStatus  int             `json:"fakeStatus"`
	MissingKeys []string        `json:"missingKeys,omitempty"`
	ExtraKeys   []string        `json:"extraKeys,omitempty"`
}

// CompareRecords compares golden (live) recordings against current (fake) recordings.
// It matches pairs by (Method, Path) and checks:
//   - Status codes match
//   - Top-level JSON keys match (structure, not values)
//
// Returns a slice of comparison results and whether all comparisons passed.
func CompareRecords(golden, current []RequestResponsePair) ([]CompareResult, bool) {
	// Index current records by method+path for lookup
	type key struct{ method, path string }
	currentByKey := make(map[key]RecordedResponse)
	for _, p := range current {
		currentByKey[key{p.Request.Method, p.Request.Path}] = p.Response
	}

	var results []CompareResult
	allPass := true

	for _, g := range golden {
		k := key{g.Request.Method, g.Request.Path}
		fake, found := currentByKey[k]
		if !found {
			results = append(results, CompareResult{
				Request:     g.Request,
				StatusMatch: false,
				KeysMatch:   false,
				LiveStatus:  g.Response.StatusCode,
				FakeStatus:  0,
				MissingKeys: []string{"(no matching request in fake server)"},
			})
			allPass = false
			continue
		}

		statusMatch := g.Response.StatusCode == fake.StatusCode

		// Compare top-level JSON keys
		liveKeys := topLevelKeys(g.Response.Body)
		fakeKeys := topLevelKeys(fake.Body)
		missing, extra := diffKeys(liveKeys, fakeKeys)
		keysMatch := len(missing) == 0

		cr := CompareResult{
			Request:     g.Request,
			StatusMatch: statusMatch,
			KeysMatch:   keysMatch,
			LiveStatus:  g.Response.StatusCode,
			FakeStatus:  fake.StatusCode,
			MissingKeys: missing,
			ExtraKeys:   extra,
		}
		results = append(results, cr)

		if !statusMatch || !keysMatch {
			allPass = false
		}
	}

	return results, allPass
}

// topLevelKeys returns sorted top-level keys from a JSON object.
func topLevelKeys(data json.RawMessage) []string {
	if len(data) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// diffKeys returns keys in "expected" but not in "actual" (missing),
// and keys in "actual" but not in "expected" (extra).
func diffKeys(expected, actual []string) (missing, extra []string) {
	eSet := make(map[string]struct{}, len(expected))
	for _, k := range expected {
		eSet[k] = struct{}{}
	}
	aSet := make(map[string]struct{}, len(actual))
	for _, k := range actual {
		aSet[k] = struct{}{}
	}
	for _, k := range expected {
		if _, ok := aSet[k]; !ok {
			missing = append(missing, k)
		}
	}
	for _, k := range actual {
		if _, ok := eSet[k]; !ok {
			extra = append(extra, k)
		}
	}
	return
}
