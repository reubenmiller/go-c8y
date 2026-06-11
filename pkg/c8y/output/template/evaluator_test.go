package template_test

import (
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatorBindsParsedJSONDocument(t *testing.T) {
	eval, err := template.NewEvaluator("{id: output.id, n: output.count + 1}", "")
	require.NoError(t, err)

	out, err := eval.Evaluate([]byte(`{"id":"1","count":2}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"1","n":3}`, out)
}

func TestEvaluatorBindsNonJSONDocumentAsString(t *testing.T) {
	eval, err := template.NewEvaluator("std.asciiUpper(output)", "")
	require.NoError(t, err)

	out, err := eval.Evaluate([]byte("not json"))
	require.NoError(t, err)
	assert.Equal(t, `"NOT JSON"`, strings.TrimSpace(out))
}

func TestEvaluatorHeaderAndExternalVariables(t *testing.T) {
	header := "local request = std.extVar('request');\nlocal double(x) = x * 2;"
	eval, err := template.NewEvaluator("{path: request.path, n: double(output.count)}", header)
	require.NoError(t, err)

	// External variables can be rebound between evaluations of the same
	// compiled template.
	eval.SetCode("request", `{"path":"/a"}`)
	out, err := eval.Evaluate([]byte(`{"count":1}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"path":"/a","n":2}`, out)

	eval.SetCode("request", `{"path":"/b"}`)
	out, err = eval.Evaluate([]byte(`{"count":3}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"path":"/b","n":6}`, out)
}

func TestEvaluatorInvalidTemplate(t *testing.T) {
	_, err := template.NewEvaluator("{invalid", "")
	assert.Error(t, err)
}
