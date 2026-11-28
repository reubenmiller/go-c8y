package dataprep

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// TestService runs smart function tests on the Data Preparation engine.
type TestService struct {
	svc httpcore.Service
}

// RunTestsOptions selects the code and the tests to run. With Package,
// the code and tests come from the package; JSCode and Tests override them
// when set.
type RunTestsOptions struct {
	Package *RulePackage

	// JSCode is the bundled smart function. It must export
	// onMessage(msg, context).
	JSCode string
	// Config is passed to the smart function as the rule configuration.
	// Defaults to an empty object.
	Config map[string]any
	// Tests are the named tests to run. Only the inputs are sent; the
	// expected outputs are checked by CompareResults.
	Tests map[string]TestCase
}

type runTestsRequest struct {
	Config map[string]any        `json:"config"`
	JSCode string                `json:"jsCode"`
	Tests  map[string]testInputs `json:"tests"`
}

type testInputs struct {
	Inputs []TestInput `json:"inputs"`
}

func (o RunTestsOptions) request() (runTestsRequest, error) {
	code, tests := o.JSCode, o.Tests
	if o.Package != nil {
		if code == "" {
			code = string(o.Package.SmartFunction)
		}
		if tests == nil {
			tests = o.Package.Tests
		}
	}
	if code == "" {
		return runTestsRequest{}, fmt.Errorf("no smart function code: set JSCode or Package")
	}
	if len(tests) == 0 {
		return runTestsRequest{}, fmt.Errorf("no tests to run")
	}
	req := runTestsRequest{Config: o.Config, JSCode: code, Tests: map[string]testInputs{}}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	for name, tc := range tests {
		for i, in := range tc.Inputs {
			if err := validateInput(in); err != nil {
				return runTestsRequest{}, fmt.Errorf("test %q input %d: %w", name, i, err)
			}
		}
		req.Tests[name] = testInputs{Inputs: tc.Inputs}
	}
	return req, nil
}

// Run runs the tests against the smart function (POST /v1/run-tests) and
// returns the outputs, logs and errors per input. A test whose code throws
// still returns StatusOK: check result.Data.Passed(), or use
// CompareResults to also check expected outputs.
func (s *TestService) Run(ctx context.Context, opt RunTestsOptions) op.Result[jsonmodels.DataPrepTestResults] {
	body, err := opt.request()
	if err != nil {
		return op.Failed[jsonmodels.DataPrepTestResults](err, false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiRunTests)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDataPrepTestResults)
}

// TestFailure describes one failed test or input.
type TestFailure struct {
	Test string
	// InputIndex is the failing input, or -1 when the failure is about the
	// test as a whole (for example the number of outputs).
	InputIndex int
	Reason     string
	// Logs are the console.log lines of the failing input.
	Logs []string
}

// TestFailedError is returned when smart function tests fail.
type TestFailedError struct {
	Failures []TestFailure
}

func (e *TestFailedError) Error() string {
	parts := make([]string, 0, len(e.Failures))
	for _, f := range e.Failures {
		if f.InputIndex >= 0 {
			parts = append(parts, fmt.Sprintf("%s[%d]: %s", f.Test, f.InputIndex, f.Reason))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", f.Test, f.Reason))
		}
	}
	return fmt.Sprintf("%d smart function test failure(s): %s", len(e.Failures), strings.Join(parts, "; "))
}

// CompareResults checks test results against the tests that produced them.
// An input that threw is always a failure. When a test has
// ExpectedOutput, the outputs of all its inputs, in order, must match it:
// the same number of objects, and each expected object must be a subset of
// the actual one (fields the expected object does not mention are
// ignored).
func CompareResults(results jsonmodels.DataPrepTestResults, tests map[string]TestCase) []TestFailure {
	var failures []TestFailure
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for k := range tests {
			if !yield(k) {
				return
			}
		}
	}) {
		tc := tests[name]
		perInput := results.Test(name)
		if len(perInput) == 0 {
			failures = append(failures, TestFailure{Test: name, InputIndex: -1, Reason: "no result returned"})
			continue
		}
		var outputs [][]byte
		threw := false
		for i, res := range perInput {
			if res.Failed() {
				threw = true
				failures = append(failures, TestFailure{Test: name, InputIndex: i, Reason: "threw: " + res.ErrorMessage(), Logs: res.Logs()})
				continue
			}
			for out := range res.Outputs() {
				outputs = append(outputs, out.Bytes())
			}
		}
		if threw || tc.ExpectedOutput == nil {
			continue
		}
		if len(outputs) != len(tc.ExpectedOutput) {
			failures = append(failures, TestFailure{Test: name, InputIndex: -1,
				Reason: fmt.Sprintf("got %d outputs, expected %d", len(outputs), len(tc.ExpectedOutput))})
			continue
		}
		for i, want := range tc.ExpectedOutput {
			if !op.DesiredStateMatches(want, outputs[i]) {
				wantJSON, _ := json.Marshal(want)
				failures = append(failures, TestFailure{Test: name, InputIndex: -1,
					Reason: fmt.Sprintf("output %d does not match: expected %s, got %s", i, wantJSON, outputs[i])})
			}
		}
	}
	return failures
}

// TestAndDeployOptions describes a package to test and deploy.
type TestAndDeployOptions struct {
	Package *RulePackage
	// CompareExpected also checks each test's ExpectedOutput. Without it,
	// a test passes when no input throws.
	CompareExpected bool
	// Config is passed to the smart function during the tests.
	Config map[string]any
	Wait   WaitOptions
}

// TestAndDeploy runs the package's tests and deploys it only if they pass.
// Failing tests return a *TestFailedError and nothing is deployed.
//
// Status: as Deploy.
func (s *RuleService) TestAndDeploy(ctx context.Context, name string, opt TestAndDeployOptions) op.Result[jsonmodels.DataPrepRule] {
	if opt.Package == nil {
		return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("a package is required"), false)
	}
	if err := opt.Package.Validate(); err != nil {
		return op.Failed[jsonmodels.DataPrepRule](err, false)
	}
	meta := map[string]any{"operation": "testAndDeploy", "rule": name, "tests": len(opt.Package.Tests)}
	return composite(ctx, meta, func(ctx context.Context) op.Result[jsonmodels.DataPrepRule] {
		if len(opt.Package.Tests) > 0 {
			run := s.tests.Run(ctx, RunTestsOptions{Package: opt.Package, Config: opt.Config})
			if run.Err != nil {
				return op.Failed[jsonmodels.DataPrepRule](run.Err, false)
			}
			tests := opt.Package.Tests
			if !opt.CompareExpected {
				tests = withoutExpected(tests)
			}
			if failures := CompareResults(run.Data, tests); len(failures) > 0 {
				return op.Failed[jsonmodels.DataPrepRule](&TestFailedError{Failures: failures}, false)
			}
		}
		return s.Deploy(ctx, name, DeployOptions{Package: opt.Package, Wait: opt.Wait})
	})
}

func withoutExpected(tests map[string]TestCase) map[string]TestCase {
	out := make(map[string]TestCase, len(tests))
	for k, v := range tests {
		v.ExpectedOutput = nil
		out[k] = v
	}
	return out
}
