package jsonmodels

import (
	"iter"
	"slices"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// DataPrepRule is a Data Preparation rule as returned by
// GET /service/dataprep/v1/rules (config and deploymentStatus) and
// GET /v1/rules/{name}/deployed or /draft (which add jsCode and tests).
type DataPrepRule struct {
	jsondoc.Facade
}

func NewDataPrepRule(b []byte) DataPrepRule {
	return DataPrepRule{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Name returns the rule name. The API keys rules by name rather than
// including it in the rule, so the SDK adds it.
func (r DataPrepRule) Name() string        { return r.Get("name").String() }
func (r DataPrepRule) Description() string { return r.Get("config.description").String() }
func (r DataPrepRule) Tags() []string      { return dpStrings(r.Get("config.tags")) }
func (r DataPrepRule) Disabled() bool      { return r.Get("config.disabled").Bool() }

// Transport, TopicPattern, ClientIDPattern and TransportID read
// config.source, which holds the "input" block of the package's
// data-prep.yaml.
func (r DataPrepRule) Transport() string    { return r.Get("config.source.transport").String() }
func (r DataPrepRule) TopicPattern() string { return r.Get("config.source.topicPattern").String() }
func (r DataPrepRule) ClientIDPattern() string {
	return r.Get("config.source.clientIDPattern").String()
}
func (r DataPrepRule) TransportID() string { return r.Get("config.source.transportID").String() }

// JSCode returns the bundled smart function. Only present in the response
// of Get and GetDraft.
func (r DataPrepRule) JSCode() string { return r.Get("jsCode").String() }

// HasTests reports whether the response includes the rule's tests (Get and
// GetDraft).
func (r DataPrepRule) HasTests() bool { return r.Get("tests").IsObject() }

// TestNames returns the names of the stored tests, sorted.
func (r DataPrepRule) TestNames() []string {
	var names []string
	r.Get("tests").ForEach(func(k, _ gjson.Result) bool {
		names = append(names, k.String())
		return true
	})
	slices.Sort(names)
	return names
}

// Test returns the raw JSON of one stored test ({"inputs": [...],
// "expectedOutput": [...]}).
func (r DataPrepRule) Test(name string) jsondoc.JSONDoc {
	return jsondoc.New([]byte(r.Get("tests." + dpEscape(name)).Raw))
}

// DeploymentStatus returns the deployment state of the rule.
func (r DataPrepRule) DeploymentStatus() DataPrepDeploymentStatus {
	return DataPrepDeploymentStatus{r.Get("deploymentStatus")}
}

// DataPrepDeploymentStatus is the deploymentStatus of a rule.
type DataPrepDeploymentStatus struct {
	v gjson.Result
}

// InProgress reports whether a deployment is still running.
func (s DataPrepDeploymentStatus) InProgress() bool { return s.v.Get("deploymentInProgress").Bool() }

// Disabled reports whether the rule is deployed in the disabled state.
func (s DataPrepDeploymentStatus) Disabled() bool { return s.v.Get("deploymentDisabled").Bool() }

// Version is an opaque identifier of the deployed version.
func (s DataPrepDeploymentStatus) Version() string { return s.v.Get("deploymentVersion").String() }

// Error is the deployment error, or empty when the last deployment worked.
func (s DataPrepDeploymentStatus) Error() string   { return s.v.Get("error").String() }
func (s DataPrepDeploymentStatus) Time() time.Time { return s.v.Get("time").Time() }
func (s DataPrepDeploymentStatus) User() string    { return s.v.Get("user").String() }

// DataPrepTestResults is the response of the run-tests endpoint: for each
// test name, one result per input message, in input order.
type DataPrepTestResults struct {
	jsondoc.Facade
}

func NewDataPrepTestResults(b []byte) DataPrepTestResults {
	return DataPrepTestResults{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Names returns the test names, sorted.
func (r DataPrepTestResults) Names() []string {
	var names []string
	r.Get().ForEach(func(k, _ gjson.Result) bool {
		names = append(names, k.String())
		return true
	})
	slices.Sort(names)
	return names
}

// Test returns the per-input results of one test.
func (r DataPrepTestResults) Test(name string) []DataPrepTestResult {
	var out []DataPrepTestResult
	for _, item := range r.Get(dpEscape(name)).Array() {
		out = append(out, DataPrepTestResult{jsondoc.Facade{JSONDoc: jsondoc.New([]byte(item.Raw))}})
	}
	return out
}

// All yields every test name (sorted) with its per-input results.
func (r DataPrepTestResults) All() iter.Seq2[string, []DataPrepTestResult] {
	return func(yield func(string, []DataPrepTestResult) bool) {
		for _, name := range r.Names() {
			if !yield(name, r.Test(name)) {
				return
			}
		}
	}
}

// Passed reports whether no input made the smart function throw. It does
// not compare outputs; see dataprep.CompareResults for that.
func (r DataPrepTestResults) Passed() bool {
	for _, results := range r.All() {
		for _, res := range results {
			if res.Failed() {
				return false
			}
		}
	}
	return true
}

// DataPrepTestResult is the result of one input message.
type DataPrepTestResult struct {
	jsondoc.Facade
}

// Failed reports whether onMessage threw for this input.
func (r DataPrepTestResult) Failed() bool { return r.Get("error").Exists() }

// ErrorMessage and ErrorStack describe the JavaScript exception, when
// Failed.
func (r DataPrepTestResult) ErrorMessage() string { return r.Get("error.message").String() }
func (r DataPrepTestResult) ErrorStack() string   { return r.Get("error.stack").String() }

// Logs returns the lines written with console.log.
func (r DataPrepTestResult) Logs() []string { return dpStrings(r.Get("logs")) }

// Outputs yields the Cumulocity objects onMessage returned.
func (r DataPrepTestResult) Outputs() iter.Seq[DataPrepOutput] {
	return dpSeq(r.Get("outputs"), func(b []byte) DataPrepOutput {
		return DataPrepOutput{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
	})
}

// DataPrepOutput is one object returned by a smart function.
type DataPrepOutput struct {
	jsondoc.Facade
}

// CumulocityType is the kind of object, for example "measurement" or
// "event".
func (o DataPrepOutput) CumulocityType() string { return o.Get("cumulocityType").String() }

// Payload is the object to create.
func (o DataPrepOutput) Payload() jsondoc.JSONDoc {
	return jsondoc.New([]byte(o.Get("payload").Raw))
}

// ExternalSourceIDs returns the external ids the object is created for.
func (o DataPrepOutput) ExternalSourceIDs() []string {
	var ids []string
	for _, item := range o.Get("externalSource").Array() {
		ids = append(ids, item.Get("externalId").String())
	}
	return ids
}

// The helpers below are private to the Data Preparation models.

func dpStrings(arr gjson.Result) []string {
	if !arr.IsArray() {
		return nil
	}
	items := arr.Array()
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.String())
	}
	return out
}

func dpSeq[T any](arr gjson.Result, ctor func([]byte) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range arr.Array() {
			if !yield(ctor([]byte(item.Raw))) {
				return
			}
		}
	}
}

// dpEscape escapes gjson path characters so names are matched literally.
func dpEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', '*', '?', '|', '#', '@', '\\', '!', '=', '<', '>', '%':
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}
