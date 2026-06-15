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
