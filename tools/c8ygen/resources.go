package main

// This file is the in-code precursor to the OpenAPI overlay (x-c8y-sdk-* extensions)
// described in docs/API_GEN.md §5. It declares, per resource, how the generated Layer-0
// substrate maps onto the hand-written package: which operation becomes which option
// struct, which params are resolver-overridden (and therefore omitted so the human can
// add an ergonomic field), and which Go types to pin for enum/ref fields.
//
// Phase 3 replaces this registry by reading the same intent from the spec/overlay.

// paginationParams are supplied by pagination.PaginationOptions and never emitted into a
// generated option struct.
var paginationParams = map[string]bool{
	"currentPage":       true,
	"pageSize":          true,
	"withTotalPages":    true,
	"withTotalElements": true,
}

// resource describes a single hand-written package and the generated artefacts it composes.
type resource struct {
	Pkg     string       // Go package name and output sub-directory under pkg/c8y/api
	Options []optionSpec // option structs to generate (query params)
	Models  []modelSpec  // façade models to generate
}

// optionSpec generates one query-parameter option struct from an operation. The struct
// shape is owned by Layer 0 (this generator); the resolver *behaviour* stays in the
// hand-written service method. Type overrides express the deliberate API-vs-SDK
// divergences (e.g. a resolver-typed Source, typed enum slices).
//
// We generate the full public struct rather than an embedded sub-struct: Go does not
// allow setting promoted fields in a composite literal, so embedding a generated params
// struct would break ListOptions{Severity: ...} for callers. See docs/API_GEN.md §6.
type optionSpec struct {
	TypeName  string            // public struct name, e.g. "ListOptions"
	Doc       string            // type doc comment (without the leading name)
	Path      string            // OAS path
	Method    string            // HTTP method (GET, PUT, ...)
	FieldType map[string]string // param name -> Go type override (e.g. "[]model.AlarmStatus")
	FieldDoc  map[string]string // param name -> doc-comment override
	Embeds    []embedSpec       // structs to embed (e.g. pagination.PaginationOptions)
	Imports   []string          // extra imports the overrides require
	Extra     []extraField      // fields NOT in the OAS (server supports them, spec omits them)
}

// extraField is a struct field that is not derived from the OAS — declared in the overlay
// for params the server accepts but the vendored spec omits. Use sparingly and document
// why each one is needed.
type extraField struct {
	Name string `yaml:"name"` // Go field name, e.g. "Revert"
	Type string `yaml:"type"` // Go type, e.g. "bool"
	Tag  string `yaml:"tag"`  // url tag body, e.g. "revert,omitempty"
	Doc  string `yaml:"doc"`  // doc comment (should explain why it is not in the OAS)
}

// embedSpec is an anonymously-embedded struct in a generated option type.
type embedSpec struct {
	Import string `yaml:"import"` // import path, e.g. ".../pkg/c8y/api/pagination"
	Type   string `yaml:"type"`   // qualified type, e.g. "pagination.PaginationOptions"
}

// modelSpec generates façade accessors for a response schema.
type modelSpec struct {
	TypeName  string          // existing façade type, e.g. "Alarm"
	Schema    string          // components/schemas key, e.g. "alarm"
	SkipProps map[string]bool // properties kept hand-written (nested/non-derivable)
}
