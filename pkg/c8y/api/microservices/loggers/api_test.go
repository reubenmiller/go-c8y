package loggers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestList verifies the loggers map is plucked from the actuator response and
// rendered as a single document (it is keyed by logger name, not an array).
func TestList(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"levels":["OFF","INFO"],"loggers":{"ROOT":{"configuredLevel":"INFO","effectiveLevel":"INFO"}}}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), "my-svc")
	if res.Err != nil {
		t.Fatalf("List: %v", res.Err)
	}
	if path != "/service/my-svc/loggers" {
		t.Errorf("path = %q, want /service/my-svc/loggers", path)
	}
	count := 0
	var raw string
	for item, err := range res.Items() {
		if err != nil {
			t.Fatalf("item err: %v", err)
		}
		count++
		raw = item.Get("ROOT.configuredLevel").String()
	}
	if count != 1 {
		t.Errorf("items = %d, want 1 (the loggers map)", count)
	}
	if raw != "INFO" {
		t.Errorf("ROOT.configuredLevel = %q, want INFO", raw)
	}
}

// TestGet verifies a single logger's level is fetched by name.
func TestGet(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"configuredLevel":"DEBUG","effectiveLevel":"DEBUG"}`))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "my-svc", "org.example")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if path != "/service/my-svc/loggers/org.example" {
		t.Errorf("path = %q, want .../loggers/org.example", path)
	}
	if res.Data.ConfiguredLevel() != "DEBUG" {
		t.Errorf("configuredLevel = %q, want DEBUG", res.Data.ConfiguredLevel())
	}
}

// TestSet verifies the log level is POSTed with the configuredLevel body. The
// same path with {"configuredLevel":null} backs the CLI delete (reset) command.
func TestSet(t *testing.T) {
	var method, path string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Set(context.Background(), "my-svc", "org.example", map[string]any{"configuredLevel": "DEBUG"})
	if res.Err != nil {
		t.Fatalf("Set: %v", res.Err)
	}
	if method != http.MethodPost || path != "/service/my-svc/loggers/org.example" {
		t.Errorf("got %s %s, want POST .../loggers/org.example", method, path)
	}
	if body["configuredLevel"] != "DEBUG" {
		t.Errorf("configuredLevel = %v, want DEBUG", body["configuredLevel"])
	}
}
