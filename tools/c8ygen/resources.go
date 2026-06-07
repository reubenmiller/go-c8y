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

// resource describes a single hand-written package and the generated artifacts it composes.
type resource struct {
	Pkg     string       // Go package name and output sub-directory under pkg/c8y/api
	Options []optionSpec // option structs to generate (query params)
	Models  []modelSpec  // façade models to generate
}

// optionSpec generates one query-parameter option struct from an operation. The struct
// shape is owned by Layer 0 (this generator); the resolver *behavior* stays in the
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
}

// embedSpec is an anonymously-embedded struct in a generated option type.
type embedSpec struct {
	Import string // import path, e.g. ".../pkg/c8y/api/pagination"
	Type   string // qualified type, e.g. "pagination.PaginationOptions"
}

// modelSpec generates façade accessors for a response schema.
type modelSpec struct {
	TypeName  string          // existing façade type, e.g. "Alarm"
	Schema    string          // components/schemas key, e.g. "alarm"
	SkipProps map[string]bool // properties kept hand-written (nested/non-derivable)
}

// pilotResources is the Phase-2 pilot: the alarms resource only.
var pilotResources = []resource{
	{
		Pkg: "alarms",
		Options: []optionSpec{
			{
				TypeName: "ListOptions",
				Doc: "ListOptions to use when searching for alarms.\n\n" +
					"The struct shape is generated from the OpenAPI spec; the Source resolver\n" +
					"field is a deliberate divergence (it accepts \"name:\"/\"ext:\"/\"query:\"\n" +
					"strings, resolved by the List method). See docs/API_GEN.md.",
				Path:   "/alarm/alarms",
				Method: "GET",
				FieldType: map[string]string{
					"status":   "[]model.AlarmStatus",
					"severity": "[]model.AlarmSeverity",
					"source":   "managedobjects.DeviceRef",
				},
				FieldDoc: map[string]string{
					"source": "Source device to filter alarms by.\n" +
						"Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,\n" +
						"or cast a string variable with managedobjects.DeviceRef(id).",
				},
				Embeds: []embedSpec{
					{Import: "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination", Type: "pagination.PaginationOptions"},
				},
				Imports: []string{
					"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model",
					"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects",
				},
			},
		},
		Models: []modelSpec{
			{
				TypeName:  "Alarm",
				Schema:    "alarm",
				SkipProps: map[string]bool{"source": true}, // nested object: SourceID() stays hand-written
			},
		},
	},
}
