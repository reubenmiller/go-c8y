package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultOverlayPath is the SDK overlay vendored into the repository.
const DefaultOverlayPath = "docs/c8y-oas.overlay.yml"

// The overlay file carries the SDK-specific codegen directives that layer on top of the
// upstream OpenAPI spec — which operation becomes which option struct, the deliberate
// type/doc divergences, embeds, and which schemas map to façade models. It is kept
// separate from docs/c8y-oas.yml so it survives `task fetch-spec`. See docs/API_GEN.md §5.
//
// This is a pragmatic, operation-keyed overlay rather than full OpenAPI Overlay 1.0
// (JSONPath actions); the structure below is intentionally close to the generator's
// internal model so the mapping is obvious.

type overlayFile struct {
	Resources []overlayResource `yaml:"resources"`
	Drift     driftWaivers      `yaml:"drift"`
}

// driftWaivers declares known-acceptable drift between the OAS and the SDK, so the
// `lint --strict` CI gate fails only on NEW, undeclared drift. Patterns are matched
// against normalized paths ({param} → {}); a trailing "*" is a prefix wildcard.
type driftWaivers struct {
	IgnoreMissing []string `yaml:"ignoreMissing"` // OAS paths the SDK intentionally omits / known TODOs
	IgnoreExtra   []string `yaml:"ignoreExtra"`   // SDK paths with no OAS counterpart (non-OAS features)
}

type overlayResource struct {
	Package string          `yaml:"package"`
	Options []overlayOption `yaml:"options"`
	Models  []overlayModel  `yaml:"models"`
}

type overlayOption struct {
	Type        string                  `yaml:"type"`
	Doc         string                  `yaml:"doc"`
	Path        string                  `yaml:"path"`
	Method      string                  `yaml:"method"`
	Fields      map[string]overlayField `yaml:"fields"`
	Embeds      []embedSpec             `yaml:"embeds"`
	Imports     []string                `yaml:"imports"`
	ExtraFields []extraField            `yaml:"extraFields"`
}

type overlayField struct {
	Type string `yaml:"type"`
	Doc  string `yaml:"doc"`
}

type overlayModel struct {
	Type   string   `yaml:"type"`
	Schema string   `yaml:"schema"`
	Skip   []string `yaml:"skip"`
}

// parseOverlay reads and parses the overlay file. A missing file yields a zero-value
// overlayFile and no error.
func parseOverlay(path string) (overlayFile, error) {
	if path == "" {
		path = DefaultOverlayPath
	}
	var f overlayFile
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return f, fmt.Errorf("read overlay %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return f, fmt.Errorf("parse overlay %s: %w", path, err)
	}
	return f, nil
}

// LoadDriftWaivers reads the drift-waiver section of the overlay.
func LoadDriftWaivers(path string) (driftWaivers, error) {
	f, err := parseOverlay(path)
	return f.Drift, err
}

// LoadOverlay reads the SDK overlay file and converts it into the generator's resource
// model. A missing file is not an error — it yields zero resources.
func LoadOverlay(path string) ([]resource, error) {
	f, err := parseOverlay(path)
	if err != nil {
		return nil, err
	}

	resources := make([]resource, 0, len(f.Resources))
	for _, or := range f.Resources {
		r := resource{Pkg: or.Package}
		for _, oo := range or.Options {
			spec := optionSpec{
				TypeName:  oo.Type,
				Doc:       oo.Doc,
				Path:      oo.Path,
				Method:    oo.Method,
				Embeds:    oo.Embeds,
				Imports:   oo.Imports,
				Extra:     oo.ExtraFields,
				FieldType: map[string]string{},
				FieldDoc:  map[string]string{},
			}
			for name, fld := range oo.Fields {
				if fld.Type != "" {
					spec.FieldType[name] = fld.Type
				}
				if fld.Doc != "" {
					spec.FieldDoc[name] = fld.Doc
				}
			}
			r.Options = append(r.Options, spec)
		}
		for _, om := range or.Models {
			m := modelSpec{TypeName: om.Type, Schema: om.Schema, SkipProps: map[string]bool{}}
			for _, s := range om.Skip {
				m.SkipProps[s] = true
			}
			r.Models = append(r.Models, m)
		}
		resources = append(resources, r)
	}
	return resources, nil
}
