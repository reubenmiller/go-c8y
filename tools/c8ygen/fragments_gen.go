package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateFragments emits typed fragment structs (each with a FragmentKey() method) from
// the OAS schemas named in the overlay, into pkg/c8y/api/model/zz_generated_fragments.go.
// Returns the number of fragments written and the file path.
func GenerateFragments(doc *OAS, source, root string, frags []overlayFragment) (int, string, error) {
	if len(frags) == 0 {
		return 0, "", nil
	}

	needTime := false
	var blocks []string
	for _, f := range frags {
		schema, ok := doc.Components.Schemas[f.Schema]
		if !ok {
			return 0, "", fmt.Errorf("fragment schema %q not found", f.Schema)
		}
		block, t := renderFragment(doc, f, schema)
		needTime = needTime || t
		blocks = append(blocks, block)
	}

	var out strings.Builder
	out.WriteString(generatedHeader("model", source))
	if needTime {
		out.WriteString("\nimport \"time\"\n")
	}
	out.WriteString("\n// Typed well-known custom fragments. Each implements model.Fragment\n")
	out.WriteString("// (FragmentKey), so it can be used on the write side (CreateOptions.Fragments)\n")
	out.WriteString("// and decoded on the read side (jsonmodels.GetFragment).\n")
	for _, b := range blocks {
		out.WriteString("\n")
		out.WriteString(b)
	}

	path := filepath.Join(root, "pkg", "c8y", "api", "model", "zz_generated_fragments.go")
	if err := writeFormatted(path, out.String()); err != nil {
		return 0, "", err
	}
	return len(frags), path, nil
}

// renderFragment emits one fragment struct + its FragmentKey method. Returns whether the
// struct uses time.Time (so the caller can add the import).
func renderFragment(doc *OAS, f overlayFragment, schema Schema) (string, bool) {
	needTime := false
	var b strings.Builder
	if d := cleanComment(schema.Description); d != "" {
		fmt.Fprintf(&b, "// %s %s\n", f.Type, d)
	}
	fmt.Fprintf(&b, "type %s struct {\n", f.Type)
	for _, propName := range sortedKeys(schema.Properties) {
		goType, t := fragmentFieldType(doc, schema.Properties[propName])
		needTime = needTime || t
		fmt.Fprintf(&b, "\t%s %s `json:%q`\n", pascalAll(propName), goType, propName+",omitempty")
	}
	b.WriteString("}\n")
	fmt.Fprintf(&b, "\n// FragmentKey implements model.Fragment.\n")
	fmt.Fprintf(&b, "func (%s) FragmentKey() string { return %q }\n", f.Type, f.Key)
	return b.String(), needTime
}

// fragmentFieldType maps a property schema to a Go field type. Scalars (and []string)
// use the shared scalar mapping; arrays fall back to []any and objects to map[string]any
// so no field is silently dropped.
func fragmentFieldType(doc *OAS, s Schema) (string, bool) {
	if goType, _, ok := doc.goScalarType(s); ok {
		return goType, goType == "time.Time"
	}
	resolved := doc.resolveSchema(s)
	if resolved.Type == "array" {
		return "[]any", false
	}
	return "map[string]any", false
}
