package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	common := &core.Service{Client: resty.New().SetBaseURL(ts.URL)}
	return NewService(common, managedobjects.NewService(common)), ts.Close
}

// TestGet verifies the id is substituted into the event path.
func TestGet(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","type":"c8y_TestEvent"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if method != http.MethodGet {
		t.Errorf("method = %q, want GET", method)
	}
	if path != "/event/events/123" {
		t.Errorf("path = %q, want /event/events/123", path)
	}
	if res.Data.ID() != "123" {
		t.Errorf("id = %q, want 123", res.Data.ID())
	}
}

// TestCreateRawPassesRawBody verifies the CLI's pre-built body is POSTed as-is.
func TestCreateRawPassesRawBody(t *testing.T) {
	var path, method string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	})
	defer closeFn()

	raw := json.RawMessage(`{"type":"c8y_TestEvent","text":"hello","source":{"id":"12345"}}`)
	res := svc.CreateRaw(context.Background(), raw)
	if res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if method != http.MethodPost || path != "/event/events" {
		t.Errorf("create = %s %s, want POST /event/events", method, path)
	}
	if body["text"] != "hello" {
		t.Errorf("body text = %v, want passthrough", body["text"])
	}
	if src, _ := body["source"].(map[string]any); src["id"] != "12345" {
		t.Errorf("body source.id = %v, want 12345", body["source"])
	}
}

// TestCreateResolvesSource verifies a "name:" source is resolved to an id via an
// inventory lookup before the event is created with the resolved source.id.
func TestCreateResolvesSource(t *testing.T) {
	var lookupQuery string
	var eventBody map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			lookupQuery = r.URL.Query().Get("query")
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"99","name":"myDevice"}]}`))
		case r.URL.Path == "/event/events" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&eventBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"e1"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	defer closeFn()

	res := svc.Create(context.Background(), CreateOptions{
		Source: managedobjects.DeviceRef("name:myDevice"),
		Type:   "c8y_TestEvent",
		Text:   "hi",
	})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if !strings.Contains(lookupQuery, "myDevice") {
		t.Errorf("lookup query = %q, want it to filter on the device name", lookupQuery)
	}
	if src, _ := eventBody["source"].(map[string]any); src["id"] != "99" {
		t.Errorf("event source.id = %v, want resolved id 99", eventBody["source"])
	}
	if eventBody["type"] != "c8y_TestEvent" || eventBody["text"] != "hi" {
		t.Errorf("event body = %v, want type/text set", eventBody)
	}
}

// TestUpdate verifies the body is PUT to the event path.
func TestUpdate(t *testing.T) {
	var path, method string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), "123", map[string]any{"text": "updated"})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut || path != "/event/events/123" {
		t.Errorf("update = %s %s, want PUT /event/events/123", method, path)
	}
	if body["text"] != "updated" {
		t.Errorf("body text = %v, want updated", body["text"])
	}
}

// TestDelete verifies a single event is removed via DELETE.
func TestDelete(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if method != http.MethodDelete || path != "/event/events/123" {
		t.Errorf("delete = %s %s, want DELETE /event/events/123", method, path)
	}
}

// TestListPlucksEvents verifies the collection is plucked from the events
// property and the filter query parameters are sent.
func TestListPlucksEvents(t *testing.T) {
	var path string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[{"id":"e1"},{"id":"e2"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{
		Type:         "c8y_TestEvent",
		FragmentType: "c8y_IsBinary",
	})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/event/events" {
		t.Errorf("path = %q, want /event/events", path)
	}
	if query.Get("type") != "c8y_TestEvent" || query.Get("fragmentType") != "c8y_IsBinary" {
		t.Errorf("query = %v, want type/fragmentType set", query)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "e1" || ids[1] != "e2" {
		t.Errorf("ids = %v, want [e1 e2]", ids)
	}
}

// TestDeleteListSendsQueryParams verifies the collection delete sends the
// filter query parameters on a DELETE to the collection endpoint.
func TestDeleteListSendsQueryParams(t *testing.T) {
	var path, method string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		query = r.URL.Query()
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.DeleteList(context.Background(), DeleteListOptions{
		Type:         "c8y_TestEvent",
		FragmentType: "c8y_IsBinary",
	})
	if res.Err != nil {
		t.Fatalf("DeleteList: %v", res.Err)
	}
	if method != http.MethodDelete || path != "/event/events" {
		t.Errorf("deleteList = %s %s, want DELETE /event/events", method, path)
	}
	if query.Get("type") != "c8y_TestEvent" || query.Get("fragmentType") != "c8y_IsBinary" {
		t.Errorf("query = %v, want type/fragmentType set", query)
	}
}
