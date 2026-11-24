package dtm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kinds of objects in a Model file.
const (
	KindPropertyDefinition    = "PropertyDefinition"
	KindAssetDefinition       = "AssetDefinition"
	KindMeasurementDefinition = "MeasurementDefinition"
	KindEventDefinition       = "EventDefinition"
	KindAlarmDefinition       = "AlarmDefinition"
	KindAsset                 = "Asset"
)

// Model is the desired state of a digital twin: definitions and, optionally,
// assets with their hierarchy, devices and linked series. Apply makes the
// tenant match it; Plan shows what Apply would change.
type Model struct {
	PropertyDefinitions    []Definition
	MeasurementDefinitions []Definition
	EventDefinitions       []Definition
	AlarmDefinitions       []Definition
	AssetDefinitions       []Definition
	Assets                 []ModelAsset
}

// ModelAsset is an asset in a Model, identified by its external id.
type ModelAsset struct {
	// ExternalID identifies the asset (c8y_ExternalId). Required.
	ExternalID string `json:"externalId"`
	// Parent makes the asset a sub-asset of another asset. It is an asset
	// reference, for example "ext:hall-1" for an asset in the same model.
	Parent AssetRef `json:"parent,omitempty"`
	// Asset is the asset body (name, type and fragments), without the
	// external id.
	Asset map[string]any `json:"asset,omitempty"`
	// Devices are assigned to the asset. Each entry is a device reference
	// (id, "name:", "ext:<type>:<id>", "query:").
	Devices []string `json:"devices,omitempty"`
	// LinkedSeries link asset series to device series.
	LinkedSeries []ModelLink `json:"linkedSeries,omitempty"`
}

// ModelLink is a linked series in a Model. See LinkOptions.
type ModelLink struct {
	Fragment       string `json:"fragment"`
	Series         string `json:"series"`
	Label          string `json:"label,omitempty"`
	Device         string `json:"device"`
	SourceFragment string `json:"sourceFragment,omitempty"`
	SourceSeries   string `json:"sourceSeries,omitempty"`
	SourceType     string `json:"sourceType,omitempty"`
}

func (l ModelLink) options() LinkOptions {
	return LinkOptions{
		Fragment: l.Fragment, Series: l.Series, Label: l.Label,
		Device: l.Device, SourceFragment: l.SourceFragment, SourceSeries: l.SourceSeries, SourceType: l.SourceType,
	}
}

// UnmarshalJSON reads a definition, keeping fields it does not know in
// Extra.
func (d *Definition) UnmarshalJSON(b []byte) error {
	type plain Definition
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	var all map[string]any
	if err := json.Unmarshal(b, &all); err != nil {
		return err
	}
	for _, known := range []string{"identifier", "contexts", "tags", "jsonSchema", "composition"} {
		delete(all, known)
	}
	*d = Definition(p)
	if len(all) > 0 {
		d.Extra = all
	} else {
		d.Extra = nil
	}
	return nil
}

// LoadModel reads a Model from files. Each path is a file or a folder;
// folders are read recursively and only *.yaml, *.yml and *.json files are
// used. A file holds one or more objects (YAML documents separated by ---,
// or a list), and each object has a "kind":
//
//	kind: PropertyDefinition
//	identifier: maxFlow
//	contexts: [asset]
//	jsonSchema: {title: Maximum flow, type: number}
//	---
//	kind: AssetDefinition
//	identifier: Pump
//	jsonSchema: {title: Pump}
//	composition:
//	  allowedProperties: [{identifier: maxFlow}]
//	---
//	kind: Asset
//	externalId: pump-07
//	parent: ext:hall-1
//	asset: {name: Pump 07, type: Pump, maxFlow: 120}
//	devices: ["name:pump-07-controller"]
//	linkedSeries:
//	  - {fragment: c8y_Flow, series: F, device: "name:pump-07-controller", sourceType: c8y_FlowMeasurement}
//
// Definition kinds take the same fields as Definition (identifier,
// contexts, tags, jsonSchema, composition, and custom fragments); Asset
// takes the fields of ModelAsset.
func LoadModel(paths ...string) (*Model, error) {
	m := &Model{}
	for _, p := range paths {
		files, err := modelFiles(p)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			if err := m.addDocuments(f, b); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

func modelFiles(p string) ([]string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{p}, nil
	}
	var files []string
	err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json":
			files = append(files, path)
		}
		return nil
	})
	slices.Sort(files)
	return files, err
}

// addDocuments adds every object in one file. JSON is valid YAML, so one
// decoder reads both.
func (m *Model) addDocuments(file string, b []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	for i := 0; ; i++ {
		var doc any
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		items, ok := doc.([]any)
		if !ok {
			items = []any{doc}
		}
		for j, item := range items {
			obj, ok := item.(map[string]any)
			if !ok {
				if item == nil {
					continue
				}
				return fmt.Errorf("%s: document %d item %d is not an object", file, i+1, j+1)
			}
			if err := m.Add(obj); err != nil {
				return fmt.Errorf("%s: document %d item %d: %w", file, i+1, j+1, err)
			}
		}
	}
}

// Add adds one object with a "kind" field to the model.
func (m *Model) Add(obj map[string]any) error {
	kind, _ := obj["kind"].(string)
	body := make(map[string]any, len(obj))
	for k, v := range obj {
		if k != "kind" {
			body[k] = v
		}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if kind == KindAsset {
		var a ModelAsset
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&a); err != nil {
			return fmt.Errorf("asset: %w (put asset fields under \"asset:\")", err)
		}
		if a.ExternalID == "" {
			return fmt.Errorf("asset: externalId is required")
		}
		m.Assets = append(m.Assets, a)
		return nil
	}
	list := m.definitions(kind)
	if list == nil {
		return fmt.Errorf("unknown kind %q", kind)
	}
	var d Definition
	if err := json.Unmarshal(b, &d); err != nil {
		return fmt.Errorf("%s: %w", kind, err)
	}
	if d.Identifier == "" {
		return fmt.Errorf("%s: identifier is required", kind)
	}
	*list = append(*list, d)
	return nil
}

func (m *Model) definitions(kind string) *[]Definition {
	switch kind {
	case KindPropertyDefinition:
		return &m.PropertyDefinitions
	case KindMeasurementDefinition:
		return &m.MeasurementDefinitions
	case KindEventDefinition:
		return &m.EventDefinitions
	case KindAlarmDefinition:
		return &m.AlarmDefinitions
	case KindAssetDefinition:
		return &m.AssetDefinitions
	}
	return nil
}
