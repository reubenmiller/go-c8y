package notification2

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

// TestGet verifies the id is substituted into the subscription path.
func TestGet(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","subscription":"mySub"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "123")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if method != http.MethodGet || path != "/notification2/subscriptions/123" {
		t.Errorf("get = %s %s, want GET /notification2/subscriptions/123", method, path)
	}
	if res.Data.ID() != "123" {
		t.Errorf("id = %q, want 123", res.Data.ID())
	}
}

// TestListPlucksSubscriptions verifies the collection is plucked from the
// subscriptions property and the filter query parameters are sent.
func TestListPlucksSubscriptions(t *testing.T) {
	var path string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subscriptions":[{"id":"s1"},{"id":"s2"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{
		Context:      "mo",
		Subscription: "mySub",
		TypeFilter:   "c8y_Test",
	})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/notification2/subscriptions" {
		t.Errorf("path = %q, want /notification2/subscriptions", path)
	}
	if query.Get("context") != "mo" || query.Get("subscription") != "mySub" || query.Get("typeFilter") != "c8y_Test" {
		t.Errorf("query = %v, want context/subscription/typeFilter set", query)
	}
	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	if len(ids) != 2 || ids[0] != "s1" || ids[1] != "s2" {
		t.Errorf("ids = %v, want [s1 s2]", ids)
	}
}

// TestListResolvesSource verifies a "name:" source is resolved to an id via an
// inventory lookup before listing with the resolved source query parameter.
func TestListResolvesSource(t *testing.T) {
	var lookupQuery, listSource string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			lookupQuery = r.URL.Query().Get("query")
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"99","name":"myDevice"}]}`))
		case r.URL.Path == "/notification2/subscriptions" && r.Method == http.MethodGet:
			listSource = r.URL.Query().Get("source")
			_, _ = w.Write([]byte(`{"subscriptions":[]}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{Source: "name:myDevice"})
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if !strings.Contains(lookupQuery, "myDevice") {
		t.Errorf("lookup query = %q, want it to filter on the device name", lookupQuery)
	}
	if listSource != "99" {
		t.Errorf("list source = %q, want resolved id 99", listSource)
	}
}

// TestSourcePassesNumericThrough verifies a plain numeric source is used as-is,
// without an inventory lookup (so dry-run needs no server round-trip).
func TestSourcePassesNumericThrough(t *testing.T) {
	var listSource string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/inventory/managedObjects" {
			t.Errorf("numeric source must not trigger an inventory lookup")
		}
		listSource = r.URL.Query().Get("source")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subscriptions":[]}`))
	})
	defer closeFn()

	if res := svc.List(context.Background(), ListOptions{Source: "12345"}); res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if listSource != "12345" {
		t.Errorf("list source = %q, want 12345", listSource)
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

	raw := json.RawMessage(`{"context":"mo","subscription":"mySub","source":{"id":"12345"},"subscriptionFilter":{"apis":["operations"]}}`)
	res := svc.CreateRaw(context.Background(), raw)
	if res.Err != nil {
		t.Fatalf("CreateRaw: %v", res.Err)
	}
	if method != http.MethodPost || path != "/notification2/subscriptions" {
		t.Errorf("create = %s %s, want POST /notification2/subscriptions", method, path)
	}
	if body["subscription"] != "mySub" || body["context"] != "mo" {
		t.Errorf("body = %v, want passthrough of context/subscription", body)
	}
	if src, _ := body["source"].(map[string]any); src["id"] != "12345" {
		t.Errorf("body source.id = %v, want 12345", body["source"])
	}
}

// TestDelete verifies a single subscription is removed via DELETE on the id path.
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
	if method != http.MethodDelete || path != "/notification2/subscriptions/123" {
		t.Errorf("delete = %s %s, want DELETE /notification2/subscriptions/123", method, path)
	}
}

// TestDeleteBySourceResolvesSource verifies the collection delete resolves a
// "name:" source and sends context + resolved source as query parameters.
func TestDeleteBySourceResolvesSource(t *testing.T) {
	var delPath, delMethod string
	var query url.Values
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/inventory/managedObjects" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"99","name":"myDevice"}]}`))
		default:
			delPath = r.URL.Path
			delMethod = r.Method
			query = r.URL.Query()
			w.WriteHeader(http.StatusNoContent)
		}
	})
	defer closeFn()

	res := svc.DeleteBySource(context.Background(), DeleteBySourceOptions{Context: "mo", Source: "name:myDevice"})
	if res.Err != nil {
		t.Fatalf("DeleteBySource: %v", res.Err)
	}
	if delMethod != http.MethodDelete || delPath != "/notification2/subscriptions" {
		t.Errorf("delete = %s %s, want DELETE /notification2/subscriptions", delMethod, delPath)
	}
	if query.Get("context") != "mo" || query.Get("source") != "99" {
		t.Errorf("query = %v, want context=mo source=99", query)
	}
}

// TestCreateTokenRawPassesRawBody verifies the token body is POSTed as-is,
// including fields the typed TokenOptions does not model (type/signed).
func TestCreateTokenRawPassesRawBody(t *testing.T) {
	var path, method string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"abc.def.ghi"}`))
	})
	defer closeFn()

	raw := json.RawMessage(`{"subscriber":"sub1","subscription":"mySub","type":"notification","signed":true}`)
	res := svc.CreateTokenRaw(context.Background(), raw)
	if res.Err != nil {
		t.Fatalf("CreateTokenRaw: %v", res.Err)
	}
	if method != http.MethodPost || path != "/notification2/token" {
		t.Errorf("create token = %s %s, want POST /notification2/token", method, path)
	}
	if body["subscriber"] != "sub1" || body["type"] != "notification" || body["signed"] != true {
		t.Errorf("body = %v, want passthrough of subscriber/type/signed", body)
	}
	if res.Data.Token() != "abc.def.ghi" {
		t.Errorf("token = %q, want abc.def.ghi", res.Data.Token())
	}
}

// TestCreateTokenDefaultsSubscriber verifies the typed CreateToken injects the
// default subscriber when none was provided.
func TestCreateTokenDefaultsSubscriber(t *testing.T) {
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"t"}`))
	})
	defer closeFn()

	if res := svc.CreateToken(context.Background(), TokenOptions{Subscription: "mySub"}); res.Err != nil {
		t.Fatalf("CreateToken: %v", res.Err)
	}
	if body["subscriber"] != "goc8y" {
		t.Errorf("subscriber = %v, want default goc8y", body["subscriber"])
	}
}

// TestUnsubscribeSubscriber verifies the token is sent as a query parameter and
// the result body is decoded.
func TestUnsubscribeSubscriber(t *testing.T) {
	var path, method, token string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		token = r.URL.Query().Get("token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"DONE"}`))
	})
	defer closeFn()

	res := svc.UnsubscribeSubscriber(context.Background(), "my-token")
	if res.Err != nil {
		t.Fatalf("UnsubscribeSubscriber: %v", res.Err)
	}
	if method != http.MethodPost || path != "/notification2/unsubscribe" {
		t.Errorf("unsubscribe = %s %s, want POST /notification2/unsubscribe", method, path)
	}
	if token != "my-token" {
		t.Errorf("token query = %q, want my-token", token)
	}
	if res.Data.Result != "DONE" {
		t.Errorf("result = %q, want DONE", res.Data.Result)
	}
}
