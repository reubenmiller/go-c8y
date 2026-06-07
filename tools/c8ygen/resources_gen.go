package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ResourcesResult reports what the per-resource pass wrote.
type ResourcesResult struct {
	Files   []string
	Options int
	Models  int
}

// GenerateResources emits the per-resource Layer-0 artifacts (option structs, façade
// models) for every resource in the overlay, rooted at repo dir `root`.
func GenerateResources(doc *OAS, source, root string, resources []resource) (ResourcesResult, error) {
	res := ResourcesResult{}
	for _, r := range resources {
		if len(r.Options) > 0 {
			path := filepath.Join(root, "pkg", "c8y", "api", r.Pkg, "zz_generated_options.go")
			src, err := renderOptions(doc, source, r)
			if err != nil {
				return res, err
			}
			if err := writeFormatted(path, src); err != nil {
				return res, err
			}
			res.Files = append(res.Files, path)
			res.Options += len(r.Options)
		}
		for _, m := range r.Models {
			path := filepath.Join(root, "pkg", "c8y", "jsonmodels", "zz_generated_"+strings.ToLower(m.Schema)+".go")
			src, err := renderModel(doc, source, m)
			if err != nil {
				return res, err
			}
			if err := writeFormatted(path, src); err != nil {
				return res, err
			}
			res.Files = append(res.Files, path)
			res.Models++
		}
	}
	return res, nil
}

// optionField is one generated struct field.
type optionField struct {
	Name    string
	Type    string
	Tag     string
	Comment string
}

// renderOptions builds an option-struct file for a resource package.
func renderOptions(doc *OAS, source string, r resource) (string, error) {
	imports := map[string]bool{}
	var blocks []string

	for _, spec := range r.Options {
		op, found := doc.operation(spec.Path, spec.Method)
		if !found {
			return "", fmt.Errorf("operation %s %s not found in spec", spec.Method, spec.Path)
		}
		fields, fieldImports, err := optionFields(doc, spec, op)
		if err != nil {
			return "", err
		}
		for _, imp := range fieldImports {
			imports[imp] = true
		}
		for _, imp := range spec.Imports {
			// only add override imports if a field actually used an override type
			for _, f := range fields {
				if strings.Contains(f.Type, lastPathSegment(imp)+".") {
					imports[imp] = true
				}
			}
		}
		for _, e := range spec.Embeds {
			imports[e.Import] = true
		}

		var b strings.Builder
		writeDoc(&b, spec.Doc)
		fmt.Fprintf(&b, "type %s struct {\n", spec.TypeName)
		for i, f := range fields {
			if i > 0 {
				b.WriteString("\n")
			}
			if f.Comment != "" {
				for _, line := range strings.Split(f.Comment, "\n") {
					fmt.Fprintf(&b, "\t// %s\n", line)
				}
			}
			fmt.Fprintf(&b, "\t%s %s `url:%q`\n", f.Name, f.Type, f.Tag)
		}
		for _, e := range spec.Embeds {
			fmt.Fprintf(&b, "\n\t%s\n", e.Type)
		}
		b.WriteString("}\n")
		blocks = append(blocks, b.String())
	}

	var out strings.Builder
	out.WriteString(generatedHeader(r.Pkg, source))
	out.WriteString(renderImports(imports))
	for _, blk := range blocks {
		out.WriteString("\n")
		out.WriteString(blk)
	}
	return out.String(), nil
}

// optionFields resolves an operation's query parameters into generated struct fields.
func optionFields(doc *OAS, spec optionSpec, op opWithParams) ([]optionField, []string, error) {
	imports := map[string]bool{}
	fields := []optionField{}
	for _, raw := range op.Parameters {
		p := doc.resolveParam(raw)
		if p.In != "query" {
			continue
		}
		if paginationParams[p.Name] {
			continue // supplied by an embedded pagination struct
		}

		goType, urlOpts, ok := doc.goScalarType(p.Schema)
		if override, has := spec.FieldType[p.Name]; has {
			goType = override
			if urlOpts == "" {
				urlOpts = ",omitempty"
			}
		} else if !ok {
			// Unmappable parameter type without an override: skip.
			continue
		}
		if strings.Contains(goType, "time.Time") {
			imports["time"] = true
		}

		comment := cleanComment(p.Description)
		if d, has := spec.FieldDoc[p.Name]; has {
			comment = d
		}
		fields = append(fields, optionField{
			Name:    pascalAll(p.Name),
			Type:    goType,
			Tag:     p.Name + urlOpts,
			Comment: comment,
		})
	}
	return fields, sortedSet(imports), nil
}

// writeDoc emits a multi-line doc comment.
func writeDoc(b *strings.Builder, doc string) {
	if doc == "" {
		return
	}
	for _, line := range strings.Split(doc, "\n") {
		if line == "" {
			b.WriteString("//\n")
		} else {
			fmt.Fprintf(b, "// %s\n", line)
		}
	}
}

// renderModel builds a façade-accessor file for a response schema.
func renderModel(doc *OAS, source string, m modelSpec) (string, error) {
	schema, ok := doc.Components.Schemas[m.Schema]
	if !ok {
		return "", fmt.Errorf("schema %q not found", m.Schema)
	}

	needTime := false
	type acc struct {
		Method, Body, GoType, Comment string
	}
	var accessors []acc
	for _, propName := range sortedKeys(schema.Properties) {
		if m.SkipProps[propName] {
			continue
		}
		prop := doc.resolveSchema(schema.Properties[propName])
		method := pascalAll(propName)
		var goType, body string
		switch {
		case prop.Type == "string" && prop.Format == "date-time":
			goType, body, needTime = "time.Time", fmt.Sprintf("m.Get(%q).Time()", propName), true
		case prop.Type == "string":
			goType, body = "string", fmt.Sprintf("m.Get(%q).String()", propName)
		case prop.Type == "integer":
			goType, body = "int64", fmt.Sprintf("m.Get(%q).Int()", propName)
		case prop.Type == "number":
			goType, body = "float64", fmt.Sprintf("m.Get(%q).Float()", propName)
		case prop.Type == "boolean":
			goType, body = "bool", fmt.Sprintf("m.Get(%q).Bool()", propName)
		default:
			continue // objects/arrays: kept hand-written
		}
		accessors = append(accessors, acc{Method: method, Body: body, GoType: goType, Comment: cleanComment(schema.Properties[propName].Description)})
	}

	var out strings.Builder
	out.WriteString(generatedHeader("jsonmodels", source))
	if needTime {
		out.WriteString("\nimport \"time\"\n")
	}
	fmt.Fprintf(&out, "\n// Generated façade accessors for the %q schema. Nested-object and non-derivable\n", m.Schema)
	fmt.Fprintf(&out, "// accessors (e.g. SourceID) and constructors are hand-written in %s.go.\n", strings.ToLower(m.Schema))
	for _, a := range accessors {
		if a.Comment != "" {
			fmt.Fprintf(&out, "\n// %s %s\n", a.Method, lowerFirst(a.Comment))
		} else {
			out.WriteString("\n")
		}
		fmt.Fprintf(&out, "func (m %s) %s() %s { return %s }\n", m.TypeName, a.Method, a.GoType, a.Body)
	}
	return out.String(), nil
}

// opWithParams is the minimal operation view the resource pass needs.
type opWithParams struct {
	Parameters []Parameter
}

// operation looks up an operation by path + method and returns its parameters.
func (o *OAS) operation(path, method string) (opWithParams, bool) {
	item, ok := o.Paths[path]
	if !ok {
		return opWithParams{}, false
	}
	for _, mo := range item.Operations() {
		if mo.Method == strings.ToUpper(method) {
			return opWithParams{Parameters: mo.Op.Parameters}, true
		}
	}
	return opWithParams{}, false
}

// renderImports emits a goimports-style import block: standard-library packages first,
// then third-party, separated by a blank line. Returns "" for no imports.
func renderImports(imports map[string]bool) string {
	if len(imports) == 0 {
		return ""
	}
	var std, ext []string
	for _, imp := range sortedSet(imports) {
		first := strings.SplitN(imp, "/", 2)[0]
		if strings.Contains(first, ".") {
			ext = append(ext, imp)
		} else {
			std = append(std, imp)
		}
	}
	var b strings.Builder
	b.WriteString("\nimport (\n")
	for _, imp := range std {
		fmt.Fprintf(&b, "\t%q\n", imp)
	}
	if len(std) > 0 && len(ext) > 0 {
		b.WriteString("\n")
	}
	for _, imp := range ext {
		fmt.Fprintf(&b, "\t%q\n", imp)
	}
	b.WriteString(")\n")
	return b.String()
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func lastPathSegment(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

// cleanComment collapses a multi-paragraph OAS description into a short single line,
// dropping markdown/HTML noise that would render poorly in godoc.
func cleanComment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Take the first line/sentence only.
	if i := strings.IndexAny(s, "\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	// Drop obvious HTML/markdown callouts.
	if strings.Contains(s, "<") || strings.HasPrefix(s, ">") {
		return ""
	}
	return s
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	// Don't lowercase acronyms / proper nouns starting words like "ID".
	return string(r)
}
