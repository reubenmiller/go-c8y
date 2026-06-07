package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultSpecURL is the canonical location of the latest Cumulocity OpenAPI spec.
const DefaultSpecURL = "https://cumulocity.com/api/core/dist/c8y-oas.yml"

// DefaultSpecPath is the spec vendored into the repository.
const DefaultSpecPath = "docs/c8y-oas.yml"

// OAS is a minimal projection of the OpenAPI document. It deliberately models only
// the parts the generator needs (paths, schemas) rather than the full OAS surface.
type OAS struct {
	OpenAPI string `yaml:"openapi"`
	Info    struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths      map[string]PathItem `yaml:"paths"`
	Components struct {
		Schemas    map[string]Schema    `yaml:"schemas"`
		Parameters map[string]Parameter `yaml:"parameters"`
	} `yaml:"components"`
}

// PathItem holds the operations defined for a single path. Non-operation keys on a
// path item (parameters, $ref, summary) are intentionally ignored.
type PathItem struct {
	Get     *Operation `yaml:"get"`
	Put     *Operation `yaml:"put"`
	Post    *Operation `yaml:"post"`
	Delete  *Operation `yaml:"delete"`
	Patch   *Operation `yaml:"patch"`
	Head    *Operation `yaml:"head"`
	Options *Operation `yaml:"options"`
}

// Operations returns the defined (method, operation) pairs in a stable order.
func (p PathItem) Operations() []MethodOp {
	out := []MethodOp{}
	for _, m := range []struct {
		name string
		op   *Operation
	}{
		{"GET", p.Get}, {"PUT", p.Put}, {"POST", p.Post}, {"DELETE", p.Delete},
		{"PATCH", p.Patch}, {"HEAD", p.Head}, {"OPTIONS", p.Options},
	} {
		if m.op != nil {
			out = append(out, MethodOp{Method: m.name, Op: *m.op})
		}
	}
	return out
}

// MethodOp pairs an HTTP method with its operation.
type MethodOp struct {
	Method string
	Op     Operation
}

// Operation is a single API operation.
type Operation struct {
	OperationID  string    `yaml:"operationId"`
	ResourceName string    `yaml:"x-codegen-resource-name"`
	Tags         []string  `yaml:"tags"`
	Summary      string    `yaml:"summary"`
	Ignore       yaml.Node `yaml:"x-codegen-ignore"`
}

// Ignored reports whether the operation is marked to be skipped by codegen.
func (o Operation) Ignored() bool { return !o.Ignore.IsZero() }

// Parameter is a (possibly $ref'd) OpenAPI parameter.
type Parameter struct {
	Ref         string `yaml:"$ref"`
	Name        string `yaml:"name"`
	In          string `yaml:"in"`
	Description string `yaml:"description"`
	Schema      Schema `yaml:"schema"`
}

// Schema is a minimal JSON Schema projection: enough for path/enum extraction.
// $ref nodes are captured but not followed during enum extraction.
type Schema struct {
	Ref         string            `yaml:"$ref"`
	Type        string            `yaml:"type"`
	Format      string            `yaml:"format"`
	Description string            `yaml:"description"`
	Enum        []any             `yaml:"enum"`
	Properties  map[string]Schema `yaml:"properties"`
	Items       *Schema           `yaml:"items"`
}

// LoadOptions controls where the spec is read from.
type LoadOptions struct {
	Path  string // local file path; used when Fetch is false
	URL   string // remote URL; used when Fetch is true
	Fetch bool   // when true, download from URL instead of reading Path
}

// Load reads and parses the OpenAPI spec from disk or the network.
func Load(opt LoadOptions) (*OAS, []byte, error) {
	var raw []byte
	var err error
	if opt.Fetch {
		url := opt.URL
		if url == "" {
			url = DefaultSpecURL
		}
		raw, err = fetch(url)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch spec from %s: %w", url, err)
		}
	} else {
		path := opt.Path
		if path == "" {
			path = DefaultSpecPath
		}
		raw, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read spec %s: %w", path, err)
		}
	}

	var doc OAS
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse spec: %w", err)
	}
	return &doc, raw, nil
}

// fetch downloads the spec from the given URL.
func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
