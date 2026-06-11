package template

import (
	"encoding/json"
	"errors"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

// Evaluator compiles a jsonnet template once (to an AST) and evaluates it
// repeatedly. Unlike the Jsonnet stage it is not tied to a pipeline: the
// caller drives evaluation and controls per-evaluation variables via SetCode
// and SetString (read in the template through std.extVar, typically rebound
// as locals in the header).
//
// The document is bound in the template as `output`: valid JSON is bound as
// the parsed value, anything else as the raw string.
//
// An Evaluator holds a single jsonnet VM and is not safe for concurrent use.
type Evaluator struct {
	vm   *jsonnet.VM
	node ast.Node
}

// NewEvaluator compiles snippet, prefixing header — jsonnet code such as
// local bindings for helper libraries and std.extVar-backed variables —
// ahead of it. Each header binding must be terminated (e.g. "local x = 1;").
// opts configure the underlying VM, e.g. WithVar, or a function registering
// native functions.
func NewEvaluator(snippet, header string, opts ...JsonnetOption) (*Evaluator, error) {
	vm := jsonnet.MakeVM()
	for _, opt := range opts {
		opt(vm)
	}

	wrapped := "function(output_json, output_is_json) " + header +
		"\nlocal output = if output_is_json then std.parseJson(output_json) else output_json;\n(\n" + snippet + "\n)"
	node, err := jsonnet.SnippetToAST("file", wrapped)
	if err != nil {
		return nil, fmt.Errorf("template: invalid jsonnet template: %w", err)
	}
	return &Evaluator{vm: vm, node: node}, nil
}

// SetCode binds a jsonnet expression as an external variable for subsequent
// Evaluate calls, available in the template via std.extVar(name).
func (e *Evaluator) SetCode(name, code string) {
	e.vm.ExtCode(name, code)
}

// SetString binds a string external variable for subsequent Evaluate calls.
func (e *Evaluator) SetString(name, value string) {
	e.vm.ExtVar(name, value)
}

// Evaluate runs the compiled template against doc. The result is jsonnet's
// serialized output (pretty-printed, with a trailing newline); callers wanting
// compact JSON should post-process it. Errors are formatted with the VM's
// ErrorFormatter (including the stack trace), matching the snippet-based
// evaluation methods.
func (e *Evaluator) Evaluate(doc []byte) (string, error) {
	e.vm.TLAVar("output_json", string(doc))
	if json.Valid(doc) {
		e.vm.TLACode("output_is_json", "true")
	} else {
		e.vm.TLACode("output_is_json", "false")
	}
	out, err := e.vm.Evaluate(e.node)
	if err != nil {
		// vm.Evaluate returns raw interpreter errors; format them the same
		// way vm.EvaluateAnonymousSnippet does so traces are included.
		return "", errors.New(e.vm.ErrorFormatter.Format(err))
	}
	return out, nil
}
