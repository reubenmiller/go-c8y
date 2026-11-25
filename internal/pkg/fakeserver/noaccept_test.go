package fakeserver

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func doRequest(t *testing.T, fs *FakeServer, method, path, body, accept string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, fs.URL()+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("t12345/admin", "admin-pass")
	req.Header.Set("Content-Type", "application/json")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(out)
}

// Cumulocity only returns the created or updated object of a POST or PUT
// request when the request has an Accept header
func TestNoAcceptHeader_DropsBodyOfPostAndPut(t *testing.T) {
	fs := New(t)

	status, body := doRequest(t, fs, http.MethodPost, "/inventory/managedObjects", `{"name":"withAccept"}`, "application/json")
	if status != http.StatusCreated || !strings.Contains(body, `"withAccept"`) {
		t.Fatalf("POST with Accept: status=%d body=%q, want 201 with the object", status, body)
	}

	status, body = doRequest(t, fs, http.MethodPost, "/inventory/managedObjects", `{"name":"noAccept"}`, "")
	if status != http.StatusCreated || body != "" {
		t.Fatalf("POST without Accept: status=%d body=%q, want 201 and no body", status, body)
	}

	// the object is still created
	status, body = doRequest(t, fs, http.MethodGet, "/inventory/managedObjects?query=name%20eq%20'noAccept'", "", "")
	if status != http.StatusOK || !strings.Contains(body, `"noAccept"`) {
		t.Fatalf("GET without Accept: status=%d body=%q, want the created object (GET keeps its body)", status, body)
	}

	status, body = doRequest(t, fs, http.MethodPut, "/inventory/managedObjects/does-not-exist", `{"name":"x"}`, "")
	if status != http.StatusNotFound || !strings.Contains(body, "notFound") {
		t.Fatalf("PUT without Accept on a missing object: status=%d body=%q, want 404 with the error body", status, body)
	}
}
