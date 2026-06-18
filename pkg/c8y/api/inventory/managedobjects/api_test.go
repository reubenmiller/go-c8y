package managedobjects

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestGetUser verifies the device-user endpoint path/method and that the
// username + enabled state are read from the response.
func TestGetUser(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userName":"device_100","enabled":true}`))
	})
	defer closeFn()

	res := svc.GetUser(context.Background(), "100")
	if res.Err != nil {
		t.Fatalf("GetUser: %v", res.Err)
	}
	if method != http.MethodGet || path != "/inventory/managedObjects/100/user" {
		t.Errorf("getUser = %s %s", method, path)
	}
	if res.Data.UserName() != "device_100" || !res.Data.Enabled() {
		t.Errorf("user = %q enabled=%v, want device_100/true", res.Data.UserName(), res.Data.Enabled())
	}
}

// TestUpdateUser verifies the PUT to the device-user endpoint, the
// managedobjectuser content-type, and that the body is passed through.
func TestUpdateUser(t *testing.T) {
	var path, method, ct string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method, ct = r.URL.Path, r.Method, r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userName":"device_100","enabled":false}`))
	})
	defer closeFn()

	res := svc.UpdateUser(context.Background(), "100", map[string]any{"enabled": false})
	if res.Err != nil {
		t.Fatalf("UpdateUser: %v", res.Err)
	}
	if method != http.MethodPut || path != "/inventory/managedObjects/100/user" {
		t.Errorf("updateUser = %s %s", method, path)
	}
	if !strings.Contains(ct, "managedobjectuser") {
		t.Errorf("content-type = %q, want managedobjectuser", ct)
	}
	if enabled, ok := body["enabled"].(bool); !ok || enabled {
		t.Errorf("body = %v, want enabled=false", body)
	}
	if res.Data.Enabled() {
		t.Errorf("response enabled = true, want false")
	}
}

// TestListSupportedMeasurements verifies the supportedMeasurements endpoint
// path and that the fragment list is read from c8y_SupportedMeasurements.
func TestListSupportedMeasurements(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"c8y_SupportedMeasurements":["c8y_Temperature","c8y_Humidity"]}`))
	})
	defer closeFn()

	res := svc.ListSupportedMeasurements(context.Background(), "100")
	if res.Err != nil {
		t.Fatalf("ListSupportedMeasurements: %v", res.Err)
	}
	if path != "/inventory/managedObjects/100/supportedMeasurements" {
		t.Errorf("path = %q", path)
	}
	if strings.Join(res.Data.Values(), ",") != "c8y_Temperature,c8y_Humidity" {
		t.Errorf("values = %v", res.Data.Values())
	}
}

// TestListSupportedSeries verifies the supportedSeries endpoint path and that
// the series list is read from c8y_SupportedSeries.
func TestListSupportedSeries(t *testing.T) {
	var path string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"c8y_SupportedSeries":["c8y_Temperature.T"]}`))
	})
	defer closeFn()

	res := svc.ListSupportedSeries(context.Background(), "100")
	if res.Err != nil {
		t.Fatalf("ListSupportedSeries: %v", res.Err)
	}
	if path != "/inventory/managedObjects/100/supportedSeries" {
		t.Errorf("path = %q", path)
	}
	if strings.Join(res.Data.Values(), ",") != "c8y_Temperature.T" {
		t.Errorf("values = %v", res.Data.Values())
	}
}

// TestGetAvailability verifies the device-availability endpoint path/method and
// that the response is parsed.
func TestGetAvailability(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dataStatus":"AVAILABLE","connectionStatus":"CONNECTED"}`))
	})
	defer closeFn()

	res := svc.GetAvailability(context.Background(), "100")
	if res.Err != nil {
		t.Fatalf("GetAvailability: %v", res.Err)
	}
	if method != http.MethodGet || path != "/inventory/managedObjects/100/availability" {
		t.Errorf("getAvailability = %s %s", method, path)
	}
	if res.Data.DataStatus() != "AVAILABLE" {
		t.Errorf("dataStatus = %q, want AVAILABLE", res.Data.DataStatus())
	}
}
