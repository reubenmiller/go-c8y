package dataprep_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/dataprep"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

const (
	pathRules    = "/service/dataprep/v1/rules"
	pathEvents   = "/service/dataprep/v1/rules/tiny-edge-events"
	pathDeployed = "/service/dataprep/v1/rules/tiny-edge-events/deployed"
	pathRunTests = "/service/dataprep/v1/run-tests"
)

type fakeServer struct {
	t      *testing.T
	server *httptest.Server
	routes map[string]func(http.ResponseWriter, *http.Request)

	mu       sync.Mutex
	requests []*http.Request
	bodies   [][]byte
}

func newFake(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{t: t, routes: map[string]func(http.ResponseWriter, *http.Request){}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, r.Clone(context.Background()))
		f.bodies = append(f.bodies, body)
		f.mu.Unlock()
		h, ok := f.routes[r.Method+" "+r.URL.Path]
		if !ok {
			write(w, http.StatusNotFound, `{"error":"Not Found","message":"no route `+r.Method+" "+r.URL.Path+`"}`)
			return
		}
		h(w, r)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeServer) handle(method, path string, h func(http.ResponseWriter, *http.Request)) {
	f.routes[method+" "+path] = h
}

func (f *fakeServer) sent(method, path string) (reqs []*http.Request, bodies [][]byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range f.requests {
		if r.Method == method && r.URL.Path == path {
			reqs = append(reqs, r)
			bodies = append(bodies, f.bodies[i])
		}
	}
	return reqs, bodies
}

func (f *fakeServer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeServer) client() *dataprep.Service {
	return testclient.Must(api.ClientOptions{
		BaseURL:          f.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t100", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	}).DataPreparation
}

func write(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func reply(status int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) { write(w, status, body) }
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const notFound = `{"error":"Not Found","message":"Rule 'tiny-edge-events' not found","status":404}`

func wantStatus[T any](t *testing.T, r op.Result[T], want op.Status) {
	t.Helper()
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Status != want {
		t.Fatalf("status = %s, want %s", r.Status, want)
	}
}

// --- reads ---

func TestListSortsAndFilters(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodGet, pathRules, reply(http.StatusOK, fixture(t, "rules_list.json")))
	svc := f.client()

	all := svc.Rules.List(context.Background(), dataprep.ListOptions{})
	var names []string
	for r := range all.Seq() {
		names = append(names, r.Name())
	}
	if all.Err != nil || !slices.Equal(names, []string{"thermo", "tiny-edge-events", "tiny-edge-measurements"}) {
		t.Fatalf("names = %v, err = %v", names, all.Err)
	}

	filtered := svc.Rules.List(context.Background(), dataprep.ListOptions{Name: "tiny-edge-*", Tags: []string{"event"}})
	var got []string
	for r := range filtered.Seq() {
		got = append(got, r.Name()+"|"+r.TopicPattern())
	}
	if !slices.Equal(got, []string{"tiny-edge-events|te/device/*///e/*"}) {
		t.Errorf("filtered = %v", got)
	}

	for r := range all.Seq() {
		if r.Name() == "thermo" {
			st := r.DeploymentStatus()
			if st.Error() == "" || !r.Disabled() || r.ClientIDPattern() != "th-*" {
				t.Errorf("thermo = %s", r.Bytes())
			}
		}
	}
}

func TestGetAndExists(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
	f.handle(http.MethodGet, "/service/dataprep/v1/rules/missing/deployed", reply(http.StatusNotFound, notFound))
	svc := f.client()

	rule := svc.Rules.Get(context.Background(), "tiny-edge-events")
	if rule.Err != nil {
		t.Fatal(rule.Err)
	}
	if rule.Data.Name() != "tiny-edge-events" || !strings.Contains(rule.Data.JSCode(), "onMessage") {
		t.Errorf("rule = %s", rule.Data.Bytes())
	}
	if !slices.Equal(rule.Data.TestNames(), []string{"identify", "no-text-no-time"}) {
		t.Errorf("tests = %v", rule.Data.TestNames())
	}

	if res := svc.Rules.Get(context.Background(), "missing"); !core.IsNotFound(res.Err) || !strings.Contains(res.Err.Error(), "not found") {
		t.Errorf("missing err = %v", res.Err)
	}
	if ok, err := svc.Rules.Exists(context.Background(), "tiny-edge-events"); !ok || err != nil {
		t.Errorf("Exists = %v, %v", ok, err)
	}
	if ok, err := svc.Rules.Exists(context.Background(), "missing"); ok || err != nil {
		t.Errorf("Exists(missing) = %v, %v", ok, err)
	}
}

func TestDelete(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodDelete, pathEvents, reply(http.StatusNoContent, ""))
	f.handle(http.MethodDelete, pathRules+"/gone", reply(http.StatusNotFound, `{"error":"Not Found"}`))
	svc := f.client()

	wantStatus(t, svc.Rules.Delete(context.Background(), "tiny-edge-events"), op.StatusNoContent)
	wantStatus(t, svc.Rules.Delete(context.Background(), "gone"), op.StatusSkipped)
}

// --- deploy ---

func TestDeploy(t *testing.T) {
	t.Run("new rule", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusNotFound, notFound))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		pkg := samplePackage()

		res := f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{Package: pkg})
		wantStatus(t, res, op.StatusCreated)
		if res.Meta["checksum"] != pkg.Checksum() || !slices.Contains(res.Meta["package"].([]string), "transform.js") {
			t.Errorf("meta = %v", res.Meta)
		}
		reqs, bodies := f.sent(http.MethodPut, pathDeployed)
		if reqs[0].Header.Get("Content-Type") != dataprep.MimeTypeGzip {
			t.Errorf("content type = %q", reqs[0].Header.Get("Content-Type"))
		}
		sent, err := dataprep.PackageFromArchive(bytes.NewReader(bodies[0]))
		if err != nil || sent.Checksum() != pkg.Checksum() {
			t.Errorf("uploaded package differs: %v", err)
		}
	})
	t.Run("existing rule", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		wantStatus(t, f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{Package: samplePackage()}), op.StatusUpdated)
	})
	t.Run("skip exists check", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		wantStatus(t, f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{
			Archive: bytes.NewReader(must(samplePackage().Archive())), SkipExistsCheck: true,
		}), op.StatusUpdated)
		if f.count() != 1 {
			t.Errorf("sent %d requests, want 1", f.count())
		}
	})
	t.Run("invalid package sends nothing, even deferred", func(t *testing.T) {
		f := newFake(t)
		bad := samplePackage()
		bad.Config.Input.TopicPattern = "a/**"
		ctx := api.WithDeferredExecution(context.Background(), true)
		res := f.client().Rules.Deploy(ctx, "x", dataprep.DeployOptions{Package: bad})
		if !errors.Is(res.Err, dataprep.ErrInvalidPackage) || f.count() != 0 {
			t.Errorf("err = %v, requests = %d", res.Err, f.count())
		}
	})
	t.Run("deferred", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusNotFound, notFound))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		ctx := api.WithDeferredExecution(context.Background(), true)
		prepared := f.client().Rules.Deploy(ctx, "tiny-edge-events", dataprep.DeployOptions{Package: samplePackage()})
		if !prepared.IsDeferred() || f.count() != 0 || prepared.Meta["package"] == nil {
			t.Fatalf("deferred=%v requests=%d meta=%v", prepared.IsDeferred(), f.count(), prepared.Meta)
		}
		wantStatus(t, prepared.Execute(context.Background()), op.StatusCreated)
	})
	t.Run("server rejects package", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusUnprocessableEntity, `{"error":"Unprocessable Entity","message":"topicPattern contains **"}`))
		res := f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{Package: samplePackage(), SkipExistsCheck: true})
		if !core.ErrHasStatus(res.Err, 422) || !strings.Contains(res.Err.Error(), "topicPattern") {
			t.Errorf("err = %v", res.Err)
		}
	})
}

// statusRule returns a deployed rule body with the given status.
func statusRule(inProgress bool, errMsg, at string) string {
	return `{"config":{"source":{"transport":"mqtt","topicPattern":"x"}},"jsCode":"x","tests":{},` +
		`"deploymentStatus":{"deploymentInProgress":` + map[bool]string{true: "true", false: "false"}[inProgress] +
		`,"deploymentVersion":"v-` + at + `","error":"` + errMsg + `","time":"` + at + `"}}`
}

func TestDeployWait(t *testing.T) {
	wait := dataprep.WaitOptions{Timeout: 5 * time.Second, PollInterval: time.Millisecond}

	t.Run("waits for a fresh, finished status", func(t *testing.T) {
		f := newFake(t)
		var mu sync.Mutex
		gets := 0
		f.handle(http.MethodGet, pathDeployed, func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			gets++
			switch gets {
			case 1, 2: // before the upload, and the old status right after it
				write(w, http.StatusOK, statusRule(false, "", "2026-11-24T00:00:00.000Z"))
			case 3:
				write(w, http.StatusOK, statusRule(true, "", "2026-11-24T00:01:00.000Z"))
			default:
				write(w, http.StatusOK, statusRule(false, "", "2026-11-24T00:01:05.000Z"))
			}
		})
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))

		res := f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{Package: samplePackage(), Wait: wait})
		wantStatus(t, res, op.StatusUpdated)
		if v := res.Data.DeploymentStatus().Version(); v != "v-2026-11-24T00:01:05.000Z" {
			t.Errorf("version = %q, want the new deployment", v)
		}
	})
	t.Run("reports a deployment error", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, statusRule(false, "SyntaxError: Unexpected token", "2026-11-24T00:01:00.000Z")))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		res := f.client().Rules.Deploy(context.Background(), "tiny-edge-events", dataprep.DeployOptions{
			Package: samplePackage(), Wait: wait, SkipExistsCheck: true,
		})
		var depErr *dataprep.DeploymentError
		if !errors.As(res.Err, &depErr) || depErr.Message != "SyntaxError: Unexpected token" {
			t.Errorf("err = %v", res.Err)
		}
	})
	t.Run("times out", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, statusRule(true, "", "2026-11-24T00:01:00.000Z")))
		res := f.client().Rules.WaitDeployed(context.Background(), "tiny-edge-events", dataprep.WaitOptions{Timeout: 50 * time.Millisecond, PollInterval: 5 * time.Millisecond})
		if res.Err == nil || !errors.Is(res.Err, context.DeadlineExceeded) {
			t.Errorf("err = %v", res.Err)
		}
	})
}

// --- apply, export, diff ---

func TestExportAndDiff(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
	svc := f.client()

	pkg := svc.Rules.Export(context.Background(), "tiny-edge-events", dataprep.ExportOptions{})
	if pkg.Err != nil {
		t.Fatal(pkg.Err)
	}
	p := pkg.Data
	if p.Config.SmartFunctionFile != "smartFunction.js" || p.Config.Input.TopicPattern != "te/device/*///e/*" {
		t.Errorf("config = %+v", p.Config)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("exported package is invalid: %v", err)
	}
	if got := p.Tests["identify"].ExpectedOutput; len(got) != 1 {
		t.Errorf("expected output = %v", got)
	}

	rule := svc.Rules.Get(context.Background(), "tiny-edge-events").Data
	if changed := dataprep.Diff(p, rule); len(changed) != 0 {
		t.Errorf("an exported rule must match itself, changed = %v", changed)
	}

	p.Config.Tags = []string{"event", "tiny-edge"} // same set, other order
	p.SmartFunction = append(p.SmartFunction, []byte("// v2\n")...)
	delete(p.Tests, "identify")
	p.Config.Input.TransportID = "mqtt" // equals the default
	if changed := dataprep.Diff(p, rule); !slices.Equal(changed, []string{"jsCode", "tests.identify"}) {
		t.Errorf("changed = %v", changed)
	}
}

func TestApply(t *testing.T) {
	deployed := func(t *testing.T) *dataprep.RulePackage {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
		return f.client().Rules.Export(context.Background(), "tiny-edge-events", dataprep.ExportOptions{}).Data
	}

	t.Run("unchanged is skipped", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
		wantStatus(t, f.client().Rules.Apply(context.Background(), "tiny-edge-events", dataprep.ApplyOptions{Package: deployed(t)}), op.StatusSkipped)
		if reqs, _ := f.sent(http.MethodPut, pathDeployed); len(reqs) != 0 {
			t.Error("PUT sent for an unchanged rule")
		}
	})
	t.Run("changed is deployed", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, fixture(t, "rule_deployed.json")))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		p := deployed(t)
		p.Config.Description = "new"
		res := f.client().Rules.Apply(context.Background(), "tiny-edge-events", dataprep.ApplyOptions{Package: p})
		wantStatus(t, res, op.StatusUpdated)
		if changed := res.Meta["changed"].([]string); !slices.Equal(changed, []string{"config.description"}) {
			t.Errorf("changed = %v", changed)
		}
	})
	t.Run("missing is created", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusNotFound, notFound))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		wantStatus(t, f.client().Rules.Apply(context.Background(), "tiny-edge-events", dataprep.ApplyOptions{Package: samplePackage()}), op.StatusCreated)
	})
	t.Run("failed deployment is retried", func(t *testing.T) {
		failed := strings.Replace(fixture(t, "rule_deployed.json"), `"error":""`, `"error":"engine restarted"`, 1)
		f := newFake(t)
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusOK, failed))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		res := f.client().Rules.Apply(context.Background(), "tiny-edge-events", dataprep.ApplyOptions{Package: deployed(t)})
		wantStatus(t, res, op.StatusUpdated)
		if changed := res.Meta["changed"].([]string); !slices.Equal(changed, []string{"deploymentStatus.error"}) {
			t.Errorf("changed = %v", changed)
		}
	})
}

// --- tests ---

func TestRunTests(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodPost, pathRunTests, reply(http.StatusOK, fixture(t, "run_tests_response.json")))
	svc := f.client()

	res := svc.Tests.Run(context.Background(), dataprep.RunTestsOptions{Package: samplePackage()})
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	_, bodies := f.sent(http.MethodPost, pathRunTests)
	var sent map[string]any
	_ = json.Unmarshal(bodies[0], &sent)
	if sent["config"] == nil || !strings.Contains(sent["jsCode"].(string), "onMessage") {
		t.Errorf("request = %s", bodies[0])
	}
	if strings.Contains(string(bodies[0]), "expectedOutput") {
		t.Error("expectedOutput must not be sent to the engine")
	}

	r := res.Data
	if r.Passed() {
		t.Error("Passed() must be false when an input threw")
	}
	if !slices.Equal(r.Names(), []string{"Broken", "Test name 1"}) {
		t.Errorf("names = %v", r.Names())
	}
	first := r.Test("Test name 1")[0]
	var types []string
	for out := range first.Outputs() {
		types = append(types, out.CumulocityType()+":"+out.Payload().Get("Humidity.H.value").String())
	}
	if !slices.Equal(types, []string{"measurement:55"}) || !slices.Equal(first.Logs(), []string{"parsed 2 rows"}) {
		t.Errorf("outputs = %v logs = %v", types, first.Logs())
	}
	if broken := r.Test("Broken")[0]; !broken.Failed() || !strings.Contains(broken.ErrorMessage(), "not a function") {
		t.Errorf("broken = %s", broken.Bytes())
	}

	if res := svc.Tests.Run(context.Background(), dataprep.RunTestsOptions{JSCode: "x"}); res.Err == nil {
		t.Error("expected an error without tests")
	}
}

func TestCompareResults(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodPost, pathRunTests, reply(http.StatusOK, fixture(t, "run_tests_response.json")))
	data := f.client().Tests.Run(context.Background(), dataprep.RunTestsOptions{JSCode: "x", Tests: map[string]dataprep.TestCase{
		"t": {Inputs: []dataprep.TestInput{dataprep.JSONInput("a", "b", "{}")}},
	}}).Data

	in := []dataprep.TestInput{dataprep.JSONInput("a", "b", "{}")}
	tests := map[string]dataprep.TestCase{
		"Test name 1": {Inputs: in, ExpectedOutput: []map[string]any{{"cumulocityType": "measurement", "payload": map[string]any{"type": "SensorReadings"}}}},
	}
	if failures := dataprep.CompareResults(data, tests); len(failures) != 0 {
		t.Errorf("subset match failed: %+v", failures)
	}

	tests["Test name 1"] = dataprep.TestCase{Inputs: in, ExpectedOutput: []map[string]any{{"cumulocityType": "event"}}}
	tests["Broken"] = dataprep.TestCase{Inputs: in}
	tests["Missing"] = dataprep.TestCase{Inputs: in}
	failures := dataprep.CompareResults(data, tests)
	var reasons []string
	for _, f := range failures {
		reasons = append(reasons, f.Test)
	}
	if !slices.Equal(reasons, []string{"Broken", "Missing", "Test name 1"}) {
		t.Errorf("failures = %+v", failures)
	}

	tests["Test name 1"] = dataprep.TestCase{Inputs: in, ExpectedOutput: []map[string]any{}}
	if f := dataprep.CompareResults(data, map[string]dataprep.TestCase{"Test name 1": tests["Test name 1"]}); len(f) != 1 || !strings.Contains(f[0].Reason, "got 1 outputs, expected 0") {
		t.Errorf("count failure = %+v", f)
	}
}

func TestTestAndDeploy(t *testing.T) {
	failing := `{"reading":[{"logs":[],"error":{"message":"boom"}}],"inlet/outlet":[{"logs":[],"outputs":[]}]}`
	passing := `{"reading":[{"logs":[],"outputs":[{"cumulocityType":"measurement","payload":{}}]}],"inlet/outlet":[{"logs":[],"outputs":[]}]}`

	t.Run("failing tests block the deploy", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodPost, pathRunTests, reply(http.StatusOK, failing))
		res := f.client().Rules.TestAndDeploy(context.Background(), "tiny-edge-events", dataprep.TestAndDeployOptions{Package: samplePackage()})
		var testErr *dataprep.TestFailedError
		if !errors.As(res.Err, &testErr) || len(testErr.Failures) != 1 || testErr.Failures[0].Test != "reading" {
			t.Fatalf("err = %v", res.Err)
		}
		if reqs, _ := f.sent(http.MethodPut, pathDeployed); len(reqs) != 0 {
			t.Error("deployed despite failing tests")
		}
	})
	t.Run("passing tests deploy", func(t *testing.T) {
		f := newFake(t)
		f.handle(http.MethodPost, pathRunTests, reply(http.StatusOK, passing))
		f.handle(http.MethodGet, pathDeployed, reply(http.StatusNotFound, notFound))
		f.handle(http.MethodPut, pathDeployed, reply(http.StatusOK, ""))
		res := f.client().Rules.TestAndDeploy(context.Background(), "tiny-edge-events", dataprep.TestAndDeployOptions{Package: samplePackage(), CompareExpected: true})
		wantStatus(t, res, op.StatusCreated)
	})
}

// --- diagnostics ---

func TestDiagnosticsDownload(t *testing.T) {
	f := newFake(t)
	f.handle(http.MethodGet, "/service/dataprep/v1/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", dataprep.MimeTypeZip)
		_, _ = w.Write([]byte("PK\x03\x04zip"))
	})
	res := f.client().Diagnostics.Download(context.Background())
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	defer res.Data.Close()
	b, _ := io.ReadAll(res.Data.Reader())
	if string(b) != "PK\x03\x04zip" {
		t.Errorf("body = %q", b)
	}
	if reqs, _ := f.sent(http.MethodGet, "/service/dataprep/v1/diagnostics"); reqs[0].Header.Get("Accept") != dataprep.MimeTypeZip {
		t.Errorf("Accept = %q", reqs[0].Header.Get("Accept"))
	}
}
