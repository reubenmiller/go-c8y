package datahub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/datahub/jobs"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

func TestQuery(t *testing.T) {
	var gotPath, gotVersion string
	var body map[string]any
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotVersion = r.URL.Query().Get("version")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rows":[{"c":1},{"c":2},{"c":3}]}`))
	})
	defer closeFn()

	res := svc.Query(context.Background(), "v1", map[string]any{"sql": "SELECT 1", "limit": 10})
	if res.Err != nil {
		t.Fatalf("Query: %v", res.Err)
	}
	if gotPath != ApiSQL || gotVersion != "v1" {
		t.Errorf("query = %s ?version=%s, want %s ?version=v1", gotPath, gotVersion, ApiSQL)
	}
	if body["sql"] != "SELECT 1" {
		t.Errorf("body sql = %v", body["sql"])
	}
	n := 0
	for _, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		n++
	}
	if n != 3 {
		t.Errorf("rows = %d, want 3", n)
	}
}

func TestJobs(t *testing.T) {
	var method, gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		method, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/service/datahub/dremio/api/v3/job/job1/results" {
			_, _ = w.Write([]byte(`{"rows":[{"x":1},{"x":2}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"job1","jobState":"COMPLETED"}`))
	})
	defer closeFn()

	if res := svc.Jobs.Create(context.Background(), map[string]any{"sql": "SELECT 1"}); res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	} else if method != http.MethodPost || gotPath != "/service/datahub/dremio/api/v3/sql" {
		t.Errorf("create = %s %s", method, gotPath)
	}

	if res := svc.Jobs.Get(context.Background(), "job1"); res.Err != nil || res.Data.JobState() != "COMPLETED" {
		t.Errorf("Get = %+v err=%v", res.Data, res.Err)
	}
	if gotPath != "/service/datahub/dremio/api/v3/job/job1" {
		t.Errorf("get path = %q", gotPath)
	}

	if res := svc.Jobs.Cancel(context.Background(), "job1"); res.Err != nil {
		t.Fatalf("Cancel: %v", res.Err)
	} else if method != http.MethodPost || gotPath != "/service/datahub/dremio/api/v3/job/job1/cancel" {
		t.Errorf("cancel = %s %s", method, gotPath)
	}

	res := svc.Jobs.GetResults(context.Background(), "job1", jobs.ResultsOptions{})
	if res.Err != nil {
		t.Fatalf("GetResults: %v", res.Err)
	}
	n := 0
	for _, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		n++
	}
	if n != 2 {
		t.Errorf("results = %d, want 2", n)
	}
}

func TestListOffloadingConfigurations(t *testing.T) {
	var gotPath string
	var gotQuery map[string][]string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCEEDED","data":{"offloadingConfigurations":[{"uuid":"a"},{"uuid":"b"}]}}`))
	})
	defer closeFn()

	res := svc.ListOffloadingConfigurations(context.Background(), ListOffloadingConfigurationsOptions{Locale: "en", Limit: 10})
	if res.Err != nil {
		t.Fatalf("ListOffloadingConfigurations: %v", res.Err)
	}
	if gotPath != ApiOffloadingConfigurations {
		t.Errorf("path = %q, want %q", gotPath, ApiOffloadingConfigurations)
	}
	if gotQuery["locale"][0] != "en" || gotQuery["limit"][0] != "10" {
		t.Errorf("query = %v", gotQuery)
	}
	n := 0
	for item, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		if item.Get("uuid").String() == "" {
			t.Errorf("item missing uuid: %s", item.Raw())
		}
		n++
	}
	if n != 2 {
		t.Errorf("configurations = %d, want 2", n)
	}
}

func TestListLatestJobs(t *testing.T) {
	var gotPath string
	var gotQuery map[string][]string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCEEDED","data":[{"jobType":"COMPACTION"}]}`))
	})
	defer closeFn()

	res := svc.ListLatestJobs(context.Background(), ListLatestJobsOptions{JobType: "COMPACTION", Offset: 5})
	if res.Err != nil {
		t.Fatalf("ListLatestJobs: %v", res.Err)
	}
	if gotPath != ApiSchedulerLatestJobs {
		t.Errorf("path = %q, want %q", gotPath, ApiSchedulerLatestJobs)
	}
	if gotQuery["jobType"][0] != "COMPACTION" || gotQuery["offset"][0] != "5" {
		t.Errorf("query = %v", gotQuery)
	}
	// The report is returned as-is (one document).
	if res.Data.Get("status").String() != "SUCCEEDED" {
		t.Errorf("report = %s", res.Data.Raw())
	}
}

func TestGetTenant(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCEEDED","data":{"initialized":true,"tenantId":"t123"}}`)) //nolint:misspell // platform JSON key
	})
	defer closeFn()

	res := svc.GetTenant(context.Background())
	if res.Err != nil {
		t.Fatalf("GetTenant: %v", res.Err)
	}
	if gotPath != ApiTenant {
		t.Errorf("path = %q, want %q", gotPath, ApiTenant)
	}
	// The "data" object is unwrapped.
	if res.Data.Get("tenantId").String() != "t123" || res.Data.Get("status").Exists() {
		t.Errorf("tenant = %s", res.Data.Raw())
	}
}

func TestListUsers(t *testing.T) {
	var gotPath string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCEEDED","data":{"users":[{"name":"u1"},{"name":"u2"},{"name":"u3"}]}}`))
	})
	defer closeFn()

	res := svc.ListUsers(context.Background())
	if res.Err != nil {
		t.Fatalf("ListUsers: %v", res.Err)
	}
	if gotPath != ApiDremioUsers {
		t.Errorf("path = %q, want %q", gotPath, ApiDremioUsers)
	}
	n := 0
	for _, err := range res.Items() {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		n++
	}
	if n != 3 {
		t.Errorf("users = %d, want 3", n)
	}
}
