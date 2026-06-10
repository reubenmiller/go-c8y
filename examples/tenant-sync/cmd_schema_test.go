package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	jsonschemavalidate "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestSchemaFileUpToDate ensures the committed tenant.schema.json matches the
// Go types. Regenerate with: go run . schema -o tenant.schema.json
func TestSchemaFileUpToDate(t *testing.T) {
	generated, err := GenerateSchema()
	require.NoError(t, err)

	committed, err := os.ReadFile("tenant.schema.json")
	require.NoError(t, err)

	assert.True(t, bytes.Equal(generated, committed),
		"tenant.schema.json is out of date; regenerate with: go run . schema -o tenant.schema.json")
}

// compileSchema compiles the generated schema for validation tests
func compileSchema(t *testing.T) *jsonschemavalidate.Schema {
	t.Helper()

	generated, err := GenerateSchema()
	require.NoError(t, err)

	doc, err := jsonschemavalidate.UnmarshalJSON(bytes.NewReader(generated))
	require.NoError(t, err)

	compiler := jsonschemavalidate.NewCompiler()
	require.NoError(t, compiler.AddResource("tenant.schema.json", doc))
	schema, err := compiler.Compile("tenant.schema.json")
	require.NoError(t, err)
	return schema
}

// validateYAML validates a YAML document against the manifest schema
func validateYAML(t *testing.T, schema *jsonschemavalidate.Schema, content []byte) error {
	t.Helper()

	var value any
	require.NoError(t, yaml.Unmarshal(content, &value))
	if value == nil {
		value = map[string]any{}
	}

	// Roundtrip through JSON so numbers/types match what the validator expects
	asJSON, err := json.Marshal(value)
	require.NoError(t, err)
	instance, err := jsonschemavalidate.UnmarshalJSON(bytes.NewReader(asJSON))
	require.NoError(t, err)

	return schema.Validate(instance)
}

func TestSchemaAcceptsTemplates(t *testing.T) {
	schema := compileSchema(t)

	// The full annotated example must satisfy the schema
	example, err := os.ReadFile("tenant.example.yaml")
	require.NoError(t, err)
	assert.NoError(t, validateYAML(t, schema, example), "tenant.example.yaml must satisfy the schema")

	// The minimal init template must satisfy the schema
	assert.NoError(t, validateYAML(t, schema, []byte(minimalTemplate)), "init template must satisfy the schema")
}

func TestSchemaRejectsInvalidManifests(t *testing.T) {
	schema := compileSchema(t)

	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "unknown top-level field",
			content: "softwares: []\n",
		},
		{
			name: "source with both path and url",
			content: `
software:
  - source:
      path: ./a
      url: https://example.com/b
`,
		},
		{
			name: "source with no fields",
			content: `
software:
  - source: {}
`,
		},
		{
			name: "github repo without owner",
			content: `
firmware:
  - name: image
    source:
      github:
        repo: invalid
`,
		},
		{
			name: "tenant option missing value",
			content: `
tenantOptions:
  - category: c8y
    key: my.key
`,
		},
		{
			name: "tenant option with both value and valueFrom",
			content: `
tenantOptions:
  - category: c8y
    key: my.key
    value: "5"
    valueFrom:
      application: devicemanagement
`,
		},
		{
			name: "valueFrom with no reference",
			content: `
tenantOptions:
  - category: c8y
    key: my.key
    valueFrom: {}
`,
		},
		{
			name: "hook without run",
			content: `
hooks:
  pre:
    - name: incomplete
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, validateYAML(t, schema, []byte(tc.content)))
		})
	}
}
