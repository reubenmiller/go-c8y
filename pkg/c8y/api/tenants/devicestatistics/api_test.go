package devicestatistics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestListDaily verifies the daily endpoint path (with the tenant and date path
// parameters), the deviceId query filter, and that the collection is read from
// the "statistics" property.
func TestListDaily(t *testing.T) {
	var path, deviceID string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		deviceID = r.URL.Query().Get("deviceId")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"statistics":[{"deviceId":"10","count":3},{"deviceId":"11","count":5}]}`))
	})
	defer closeFn()

	res := svc.ListDaily(context.Background(), ListOptions{TenantID: "t123", Date: "2024-01-15", DeviceID: "500"})
	if res.Err != nil {
		t.Fatalf("ListDaily: %v", res.Err)
	}
	if path != "/tenant/statistics/device/t123/daily/2024-01-15" {
		t.Errorf("daily path = %q", path)
	}
	if deviceID != "500" {
		t.Errorf("deviceId query = %q, want 500", deviceID)
	}
	var ids []string
	for item := range op.Iter(res) {
		ids = append(ids, item.DeviceID())
	}
	if strings.Join(ids, ",") != "10,11" {
		t.Errorf("plucked deviceIds = %v, want [10 11]", ids)
	}
}

// TestListMonthly verifies the monthly endpoint path and that the deviceId
// filter is omitted when empty.
func TestListMonthly(t *testing.T) {
	var path, rawQuery string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"statistics":[{"deviceId":"20","count":7}]}`))
	})
	defer closeFn()

	res := svc.ListMonthly(context.Background(), ListOptions{TenantID: "t123", Date: "2024-01-01"})
	if res.Err != nil {
		t.Fatalf("ListMonthly: %v", res.Err)
	}
	if path != "/tenant/statistics/device/t123/monthly/2024-01-01" {
		t.Errorf("monthly path = %q", path)
	}
	if strings.Contains(rawQuery, "deviceId") {
		t.Errorf("monthly query = %q, want no deviceId filter", rawQuery)
	}
	count := 0
	for range op.Iter(res) {
		count++
	}
	if count != 1 {
		t.Errorf("items = %d, want 1", count)
	}
}
