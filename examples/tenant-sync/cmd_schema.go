package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
)

// SchemaID is the canonical URL of the published schema. Reference it from a
// manifest to get IDE completion and validation:
//
//	# yaml-language-server: $schema=<SchemaID>
const SchemaID = "https://raw.githubusercontent.com/reubenmiller/go-c8y/main/examples/tenant-sync/tenant.schema.json"

// JSONSchemaExtend enforces that exactly one of path, url or github is set,
// mirroring Source.Validate
func (Source) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.OneOf = []*jsonschema.Schema{
		{Required: []string{"path"}, Not: &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Required: []string{"url"}},
				{Required: []string{"github"}},
			},
		}},
		{Required: []string{"url"}, Not: &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Required: []string{"path"}},
				{Required: []string{"github"}},
			},
		}},
		{Required: []string{"github"}, Not: &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Required: []string{"path"}},
				{Required: []string{"url"}},
			},
		}},
	}
}

// JSONSchemaExtend enforces that exactly one of value or valueFrom is set,
// mirroring the Manifest.Validate rules
func (TenantOptionSpec) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.OneOf = []*jsonschema.Schema{
		{Required: []string{"value"}, Not: &jsonschema.Schema{Required: []string{"valueFrom"}}},
		{Required: []string{"valueFrom"}, Not: &jsonschema.Schema{Required: []string{"value"}}},
	}
}

// JSONSchemaExtend enforces that exactly one lookup reference is set
func (TenantOptionValueFrom) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.OneOf = []*jsonschema.Schema{
		{Required: []string{"application"}, Not: &jsonschema.Schema{Required: []string{"device"}}},
		{Required: []string{"device"}, Not: &jsonschema.Schema{Required: []string{"application"}}},
	}
}

// GenerateSchema builds the JSON schema for the manifest from the Go types
func GenerateSchema() ([]byte, error) {
	reflector := &jsonschema.Reflector{
		// Inline the Manifest type at the top level instead of a $ref
		ExpandedStruct: true,
	}

	schema := reflector.Reflect(&Manifest{})
	schema.ID = SchemaID
	schema.Title = "tenant-sync manifest"
	schema.Description = "Declarative description of the desired state of a Cumulocity IoT tenant, applied with the tenant-sync CLI"

	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func cmdSchema(args []string) error {
	flags := flag.NewFlagSet("schema", flag.ExitOnError)
	var (
		output = flags.String("o", "", "Write the schema to a file instead of stdout")
	)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tenant-sync schema [flags]\n\nGenerate the JSON schema for the manifest file.\n\nReference the schema from a manifest for IDE completion (VS Code YAML extension, JetBrains):\n\n  # yaml-language-server: $schema=%s\n\nFlags:\n", SchemaID)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: use -o to write to a file")
	}

	schema, err := GenerateSchema()
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	if *output == "" {
		_, err := os.Stdout.Write(schema)
		return err
	}

	if err := os.WriteFile(*output, schema, 0o644); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}
	fmt.Printf("✓ Wrote schema to %s\n", *output)
	return nil
}
