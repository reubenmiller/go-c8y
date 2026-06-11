// Package template provides response-shaping stages backed by template
// languages. The jsonnet implementation compiles the template once (to an
// AST) and evaluates it per item, unlike evaluating a snippet string per
// item which re-parses the template every time.
package template

import (
	"fmt"
	"strconv"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/tidwall/pretty"
)

// JsonnetOption configures the jsonnet stage.
type JsonnetOption func(*jsonnet.VM)

// WithVar binds an external variable (a JSON value) that is constant for the
// stream, available in the template via std.extVar(name). Use this for
// request/response metadata, flags, etc.
func WithVar(name string, jsonValue string) JsonnetOption {
	return func(vm *jsonnet.VM) {
		vm.ExtCode(name, jsonValue)
	}
}

// WithStringVar binds an external string variable, available in the template
// via std.extVar(name).
func WithStringVar(name string, value string) JsonnetOption {
	return func(vm *jsonnet.VM) {
		vm.ExtVar(name, value)
	}
}

// Jsonnet returns a stage that transforms each document with a jsonnet
// template. The template is compiled once; per item it is invoked with two
// top-level bindings:
//
//	output — the current document
//	index  — zero-based position of the document in the stream
//
// Example: {id: output.id, name: output.name, alarms: output.c8y_ActiveAlarmsStatus}
//
// The template result must be a JSON value; it is compacted before being
// passed downstream. The returned stage holds a single jsonnet VM and must
// not be shared across concurrently running pipelines.
func Jsonnet(snippet string, opts ...JsonnetOption) (output.Stage, error) {
	vm := jsonnet.MakeVM()
	for _, opt := range opts {
		opt(vm)
	}

	// Wrap the template as a function so per-item values are bound as
	// top-level arguments. The document is bound as a string and decoded
	// with std.parseJson (native Go) — binding it as jsonnet code instead
	// would make the interpreter evaluate the whole document as a program,
	// which is ~10x slower per item.
	wrapped := "function(output_json, index) local output = std.parseJson(output_json); (\n" + snippet + "\n)"
	node, err := jsonnet.SnippetToAST("template", wrapped)
	if err != nil {
		return nil, fmt.Errorf("template: invalid jsonnet template: %w", err)
	}

	index := 0
	return output.Map(func(doc jsondoc.JSONDoc) (jsondoc.JSONDoc, error) {
		vm.TLAVar("output_json", string(doc.Raw()))
		vm.TLACode("index", strconv.Itoa(index))
		index++
		out, err := vm.Evaluate(node)
		if err != nil {
			return jsondoc.Empty(), fmt.Errorf("template: evaluation failed: %w", err)
		}
		return jsondoc.New(pretty.Ugly([]byte(out))), nil
	}), nil
}
