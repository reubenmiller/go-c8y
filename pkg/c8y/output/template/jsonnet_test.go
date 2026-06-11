package template_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const body = `{"items": [
	{"id": "1", "name": "alpha", "c8y_Hardware": {"model": "RPi4"}},
	{"id": "2", "name": "beta", "c8y_Hardware": {"model": "NUC"}}
]}`

func TestJsonnetProjection(t *testing.T) {
	stage, err := template.Jsonnet(`{id: output.id, model: output.c8y_Hardware.model, position: index}`)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = output.Render(context.Background(),
		output.FromBytes([]byte(body), "items"),
		encode.NewJSONArray(&buf), stage)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"id": "1", "model": "RPi4", "position": 0},
		{"id": "2", "model": "NUC", "position": 1}
	]`, buf.String())
}

func TestJsonnetWithVars(t *testing.T) {
	stage, err := template.Jsonnet(
		`output { tenant: std.extVar('tenant') }`,
		template.WithStringVar("tenant", "t12345"),
	)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = output.Render(context.Background(),
		output.FromBytes([]byte(body), "items"),
		encode.NewJSONArray(&buf), stage)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"tenant":"t12345"`)
	assert.Contains(t, buf.String(), `"name":"alpha"`)
}

func TestJsonnetInvalidTemplate(t *testing.T) {
	_, err := template.Jsonnet(`{id: `)
	assert.Error(t, err)
}

func TestJsonnetEvaluationError(t *testing.T) {
	stage, err := template.Jsonnet(`{id: output.doesNotExist.nested}`)
	require.NoError(t, err)

	err = output.Render(context.Background(),
		output.FromBytes([]byte(body), "items"),
		encode.NewJSONArray(&bytes.Buffer{}), stage)
	assert.ErrorContains(t, err, "evaluation failed")
}
