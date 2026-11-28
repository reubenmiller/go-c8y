package dataprep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"reflect"
	"slices"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// RuleService manages Data Preparation rules.
type RuleService struct {
	svc   httpcore.Service
	tests *TestService
}

// ListOptions filters the rule list. The endpoint returns every rule in
// one response, so the filters are applied by the SDK.
type ListOptions struct {
	// Name matches rule names; * matches any characters.
	Name string
	// Tags keeps rules that have all of these tags.
	Tags []string
}

func (o ListOptions) keep(r jsonmodels.DataPrepRule) bool {
	if o.Name != "" {
		if ok, _ := path.Match(o.Name, r.Name()); !ok {
			return false
		}
	}
	tags := r.Tags()
	for _, t := range o.Tags {
		if !slices.Contains(tags, t) {
			return false
		}
	}
	return true
}

// List returns all rules with their configuration and deployment status,
// sorted by name. The API returns an object keyed by rule name; the SDK
// turns it into a collection and adds a "name" field to each rule.
func (s *RuleService) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DataPrepRule] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiRules)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), func(b []byte) jsonmodels.DataPrepRule {
		return jsonmodels.NewDataPrepRule(rulesArray(b, opt.keep))
	})
}

// rulesArray converts {"name": {...}} into a sorted [{"name": ..., ...}].
func rulesArray(b []byte, keep func(jsonmodels.DataPrepRule) bool) []byte {
	byName := gjson.ParseBytes(b)
	var names []string
	byName.ForEach(func(k, _ gjson.Result) bool {
		names = append(names, k.String())
		return true
	})
	slices.Sort(names)
	items := make([]json.RawMessage, 0, len(names))
	for _, name := range names {
		rule := withName([]byte(byName.Get(escape(name)).Raw), name)
		if keep == nil || keep(jsonmodels.NewDataPrepRule(rule)) {
			items = append(items, rule)
		}
	}
	out, _ := json.Marshal(items)
	return out
}

func withName(b []byte, name string) []byte {
	doc, err := jsondoc.New(b).Set("name", name)
	if err != nil {
		return b
	}
	return doc.Bytes()
}

// Get returns the deployed rule, including its smart function code
// (JSCode) and tests. A rule that does not exist returns a 404 error.
func (s *RuleService) Get(ctx context.Context, name string) op.Result[jsonmodels.DataPrepRule] {
	return s.get(ctx, name, apiRuleDeployed)
}

// GetDraft returns the version of the rule being edited in the Data
// Preparation application.
func (s *RuleService) GetDraft(ctx context.Context, name string) op.Result[jsonmodels.DataPrepRule] {
	return s.get(ctx, name, apiRuleDraft)
}

func (s *RuleService) get(ctx context.Context, name, url string) op.Result[jsonmodels.DataPrepRule] {
	if name == "" {
		return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("empty rule name"), false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramRuleName, name).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(url)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), func(b []byte) jsonmodels.DataPrepRule {
		return jsonmodels.NewDataPrepRule(withName(b, name))
	})
}

// Exists reports whether a rule with this name is deployed.
func (s *RuleService) Exists(ctx context.Context, name string) (bool, error) {
	res := s.Get(ctx, name)
	switch {
	case res.Err == nil:
		return true, nil
	case core.IsNotFound(res.Err):
		return false, nil
	default:
		return false, res.Err
	}
}

// Delete removes the rule, including its draft and deployment state. It
// cannot be undone. A rule that does not exist returns StatusSkipped
// without an error, so Delete is safe in defer.
func (s *RuleService) Delete(ctx context.Context, name string) op.Result[core.NoContent] {
	if name == "" {
		return op.Failed[core.NoContent](fmt.Errorf("empty rule name"), false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramRuleName, name).
		SetURL(apiRule)
	res := httpcore.ExecuteNoContent(ctx, httpcore.NewTryRequest(s.svc.Client, req), map[string]any{"rule": name})
	if core.IsNotFound(res.Err) {
		skipped := op.Skipped(core.NoContent{}, "not found")
		skipped.HTTPStatus = res.HTTPStatus
		skipped.Request = res.Request
		return skipped
	}
	return res
}

// WaitOptions controls waiting for a deployment to finish.
type WaitOptions struct {
	// Timeout is how long to wait. 0 does not wait.
	Timeout time.Duration
	// PollInterval is the time between checks. Defaults to 1s.
	PollInterval time.Duration
}

// DeployOptions describes what to deploy. Set exactly one of Package or
// Archive.
type DeployOptions struct {
	// Package is validated (see RulePackage.Validate) and archived.
	Package *RulePackage
	// Archive is a ready-made tar.gz, sent as-is.
	Archive io.Reader

	// Wait polls the rule after the upload until the deployment is no
	// longer in progress, then fails with *DeploymentError if the engine
	// reported an error. With Wait, result.Data is the deployed rule.
	Wait WaitOptions

	// SkipExistsCheck drops the GET before the upload. The status is then
	// always StatusUpdated, because the SDK cannot tell a new rule from an
	// overwritten one.
	SkipExistsCheck bool
}

func (o DeployOptions) body() ([]byte, map[string]any, error) {
	switch {
	case o.Package != nil && o.Archive != nil:
		return nil, nil, fmt.Errorf("set either Package or Archive, not both")
	case o.Package != nil:
		b, err := o.Package.Archive()
		if err != nil {
			return nil, nil, err
		}
		return b, map[string]any{
			"package":  o.Package.Entries(),
			"checksum": o.Package.Checksum(),
			"size":     len(b),
		}, nil
	case o.Archive != nil:
		b, err := io.ReadAll(o.Archive)
		if err != nil {
			return nil, nil, fmt.Errorf("read archive: %w", err)
		}
		return b, map[string]any{"size": len(b)}, nil
	default:
		return nil, nil, fmt.Errorf("set Package or Archive")
	}
}

// DeploymentError reports a deployment the engine accepted but could not
// complete (deploymentStatus.error).
type DeploymentError struct {
	Rule    string
	Message string
	Version string
}

func (e *DeploymentError) Error() string {
	return fmt.Sprintf("deployment of rule %q failed: %s", e.Rule, e.Message)
}

// Deploy uploads a rule (PUT /v1/rules/{name}/deployed), replacing a rule
// with the same name. The rule name comes from name, not from the package.
//
// The package is validated before anything is sent, also under dry run and
// deferred execution, and result.Meta lists its entries and checksum.
//
// Status: StatusCreated (new rule) or StatusUpdated.
func (s *RuleService) Deploy(ctx context.Context, name string, opt DeployOptions) op.Result[jsonmodels.DataPrepRule] {
	if name == "" {
		return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("empty rule name"), false)
	}
	body, meta, err := opt.body()
	if err != nil {
		return op.Failed[jsonmodels.DataPrepRule](err, false)
	}
	meta["operation"] = "deploy"
	meta["rule"] = name
	return composite(ctx, meta, func(ctx context.Context) op.Result[jsonmodels.DataPrepRule] {
		var previous *jsonmodels.DataPrepRule
		if !opt.SkipExistsCheck {
			res := s.Get(ctx, name)
			switch {
			case res.Err == nil:
				previous = &res.Data
			case !core.IsNotFound(res.Err):
				return res
			}
		}
		return s.deploy(ctx, name, body, opt.SkipExistsCheck || previous != nil, previous, opt.Wait)
	})
}

// composite runs fn now, or defers it under deferred execution. meta is set
// on the prepared result and on the result fn returns.
func composite[T any](ctx context.Context, meta map[string]any, fn func(context.Context) op.Result[T]) op.Result[T] {
	withMeta := func(r op.Result[T]) op.Result[T] {
		for k, v := range meta {
			if _, exists := r.Meta[k]; !exists {
				r = op.WithMeta(r, k, v)
			}
		}
		return r
	}
	return op.ExecuteOrDefer(ctx, withMeta(op.WithExecutor(op.Result[T]{}, func(ctx context.Context) op.Result[T] {
		return withMeta(fn(ctx))
	})))
}

// deploy sends the archive and optionally waits. previous is the rule
// before the upload, used to tell the new deployment status from the old.
func (s *RuleService) deploy(ctx context.Context, name string, body []byte, existed bool, previous *jsonmodels.DataPrepRule, wait WaitOptions) op.Result[jsonmodels.DataPrepRule] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(paramRuleName, name).
		SetContentType(MimeTypeGzip).
		SetBody(body).
		SetURL(apiRuleDeployed)
	put := httpcore.ExecuteNoContent(ctx, httpcore.NewTryRequest(s.svc.Client, req))
	if put.Err != nil {
		return op.WithRequest(op.WithHTTPStatus(op.Failed[jsonmodels.DataPrepRule](put.Err, false), put.HTTPStatus), put.Request)
	}

	data := jsonmodels.NewDataPrepRule(withName([]byte(`{}`), name))
	if wait.Timeout > 0 {
		var since time.Time
		if previous != nil {
			since = previous.DeploymentStatus().Time()
		}
		res := s.waitDeployed(ctx, name, since, wait)
		if res.Err != nil {
			return res
		}
		data = res.Data
	}

	result := op.Created(data)
	if existed {
		result = op.Updated(data)
	}
	return op.WithDuration(op.WithRequest(op.WithHTTPStatus(result, put.HTTPStatus), put.Request), put.Duration)
}

// WaitDeployed polls the rule until its deployment is no longer in
// progress, and fails with *DeploymentError if the deployment reported an
// error.
func (s *RuleService) WaitDeployed(ctx context.Context, name string, opt WaitOptions) op.Result[jsonmodels.DataPrepRule] {
	return s.waitDeployed(ctx, name, time.Time{}, opt)
}

// waitDeployed waits for a deployment status newer than since (zero: any).
func (s *RuleService) waitDeployed(ctx context.Context, name string, since time.Time, opt WaitOptions) op.Result[jsonmodels.DataPrepRule] {
	interval := opt.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	for {
		res := s.Get(ctx, name)
		if res.Err != nil && !errors.Is(res.Err, context.DeadlineExceeded) {
			return res
		}
		if res.Err == nil {
			st := res.Data.DeploymentStatus()
			fresh := since.IsZero() || st.Time().After(since)
			if fresh && !st.InProgress() {
				if st.Error() != "" {
					return op.Failed[jsonmodels.DataPrepRule](&DeploymentError{Rule: name, Message: st.Error(), Version: st.Version()}, false)
				}
				return res
			}
		}
		select {
		case <-ctx.Done():
			return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("rule %q: deployment did not finish within %s: %w", name, opt.Timeout, ctx.Err()), true)
		case <-time.After(interval):
		}
	}
}

// ApplyOptions describes the desired rule.
type ApplyOptions struct {
	Package *RulePackage
	Wait    WaitOptions
}

// Apply deploys the package only when it differs from the deployed rule.
// It compares the smart function code, the configuration (description,
// tags, disabled and input) and the tests. A rule whose last deployment
// failed is deployed again even if it matches.
//
// Status: StatusCreated, StatusUpdated (Meta["changed"] lists what
// differed) or StatusSkipped (no upload).
func (s *RuleService) Apply(ctx context.Context, name string, opt ApplyOptions) op.Result[jsonmodels.DataPrepRule] {
	if name == "" {
		return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("empty rule name"), false)
	}
	if opt.Package == nil {
		return op.Failed[jsonmodels.DataPrepRule](fmt.Errorf("a package is required"), false)
	}
	body, err := opt.Package.Archive()
	if err != nil {
		return op.Failed[jsonmodels.DataPrepRule](err, false)
	}
	meta := map[string]any{"operation": "apply", "rule": name, "checksum": opt.Package.Checksum()}
	return composite(ctx, meta, func(ctx context.Context) op.Result[jsonmodels.DataPrepRule] {
		current := s.Get(ctx, name)
		switch {
		case core.IsNotFound(current.Err):
			return s.deploy(ctx, name, body, false, nil, opt.Wait)
		case current.Err != nil:
			return current
		}
		changed := Diff(opt.Package, current.Data)
		failed := current.Data.DeploymentStatus().Error() != ""
		if len(changed) == 0 && !failed {
			s := op.Skipped(current.Data, "no changes detected")
			s.HTTPStatus = current.HTTPStatus
			return s
		}
		if failed {
			changed = append(changed, "deploymentStatus.error")
		}
		return op.WithMeta(s.deploy(ctx, name, body, true, &current.Data, opt.Wait), "changed", changed)
	})
}

// Diff lists what differs between a package and a deployed rule (as
// returned by Get): "jsCode", "config.description", "config.tags",
// "config.disabled", "config.source", and "tests.<name>" for each added,
// removed or changed test. An empty result means the package matches.
//
// ai-chat.yaml is not compared: the server keeps the existing chat history
// when a package has none, so it is not part of the desired state.
func Diff(p *RulePackage, rule jsonmodels.DataPrepRule) []string {
	var changed []string
	if string(p.SmartFunction) != rule.JSCode() {
		changed = append(changed, "jsCode")
	}
	c := p.Config
	if c.Description != rule.Description() {
		changed = append(changed, "config.description")
	}
	if !sameSet(c.Tags, rule.Tags()) {
		changed = append(changed, "config.tags")
	}
	if c.Disabled != rule.Disabled() {
		changed = append(changed, "config.disabled")
	}
	wantTransportID := orDefault(c.Input.TransportID, c.Input.Transport)
	haveTransportID := orDefault(rule.TransportID(), rule.Transport())
	if c.Input.Transport != rule.Transport() || c.Input.TopicPattern != rule.TopicPattern() ||
		c.Input.ClientIDPattern != rule.ClientIDPattern() || wantTransportID != haveTransportID {
		changed = append(changed, "config.source")
	}

	stored := map[string]TestCase{}
	_ = json.Unmarshal([]byte(rule.Get("tests").Raw), &stored)
	for _, name := range slices.Sorted(maps.Keys(unionKeys(p.Tests, stored))) {
		want, inPkg := p.Tests[name]
		have, onServer := stored[name]
		if !inPkg || !onServer || !sameTest(want, have) {
			changed = append(changed, "tests."+name)
		}
	}
	return changed
}

func sameTest(a, b TestCase) bool {
	return reflect.DeepEqual(normalizeTest(a), normalizeTest(b))
}

// normalizeTest round-trips a test through JSON, so values written by Go
// and values read from the server compare the same way.
func normalizeTest(tc TestCase) any {
	b, _ := json.Marshal(tc)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

func unionKeys[V any](a, b map[string]V) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		out[k] = struct{}{}
	}
	for k := range b {
		out[k] = struct{}{}
	}
	return out
}

// ExportOptions selects what to export.
type ExportOptions struct {
	// Draft exports the draft instead of the deployed rule.
	Draft bool
}

// Export turns a deployed rule (or its draft) back into a package, for
// example to write it to disk with WriteDir or deploy it to another
// tenant. The smart function is named smartFunction.js, as in the Data
// Preparation application's exports. The AI chat history is not part of
// the rule read endpoints and is not exported.
func (s *RuleService) Export(ctx context.Context, name string, opt ExportOptions) op.Result[*RulePackage] {
	res := s.Get(ctx, name)
	if opt.Draft {
		res = s.GetDraft(ctx, name)
	}
	if res.Err != nil {
		return op.WithHTTPStatus(op.Failed[*RulePackage](res.Err, false), res.HTTPStatus)
	}
	p, err := PackageFromRule(res.Data)
	if err != nil {
		return op.Failed[*RulePackage](err, false)
	}
	return op.OK(p)
}

// PackageFromRule builds a package from a rule returned by Get or GetDraft.
func PackageFromRule(rule jsonmodels.DataPrepRule) (*RulePackage, error) {
	if !rule.HasTests() && rule.JSCode() == "" {
		return nil, fmt.Errorf("rule %q has no code; use a rule returned by Get or GetDraft, not List", rule.Name())
	}
	p := NewPackage(RuleConfig{
		SmartFunctionFile: ExportedSmartFunctionFile,
		Description:       rule.Description(),
		Tags:              rule.Tags(),
		Disabled:          rule.Disabled(),
		Input: Input{
			Transport:       rule.Transport(),
			TopicPattern:    rule.TopicPattern(),
			ClientIDPattern: rule.ClientIDPattern(),
			TransportID:     rule.TransportID(),
		},
	}).WithSmartFunction(rule.JSCode())
	if raw := rule.Get("tests").Raw; raw != "" {
		if err := json.Unmarshal([]byte(raw), &p.Tests); err != nil {
			return nil, fmt.Errorf("rule %q: read tests: %w", rule.Name(), err)
		}
	}
	return p, nil
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func orDefault(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// escape escapes gjson path characters so a key is matched literally.
func escape(s string) string {
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
