package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/smartresttemplates"
)

// smartRestCollectionFile is the JSON document of an exported SmartREST 2.0
// template collection (one collection per file). Only the fields making up
// the desired state are read; platform bookkeeping fields (id, lastUpdated,
// owner, ...) present in exports are ignored.
type smartRestCollectionFile struct {
	Name       string `json:"name"`
	ExternalID string `json:"__externalId"`
	Type       string `json:"type"`

	// Templates is a pointer so a file without the fragment (i.e. not a
	// template collection export) can be reported as an error
	Templates *struct {
		RequestTemplates  []map[string]any `json:"requestTemplates"`
		ResponseTemplates []map[string]any `json:"responseTemplates"`
	} `json:"com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate"`
}

// loadSmartRestCollection reads and validates an exported collection file.
// nameOverride (from the manifest) wins over the name/__externalId in the file.
func loadSmartRestCollection(path, nameOverride string) (*smartRestCollectionFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	collection := &smartRestCollectionFile{}
	if err := json.Unmarshal(raw, collection); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if nameOverride != "" {
		collection.Name = nameOverride
	} else if collection.Name == "" {
		collection.Name = collection.ExternalID
	}

	if collection.Type != "" && collection.Type != smartresttemplates.ManagedObjectType {
		return nil, fmt.Errorf("not a SmartREST 2.0 template collection (type %q, expected %q)", collection.Type, smartresttemplates.ManagedObjectType)
	}
	if collection.Templates == nil {
		return nil, fmt.Errorf("not a SmartREST 2.0 template collection (missing %s fragment)", smartresttemplates.FragmentTemplates)
	}
	if collection.Name == "" {
		return nil, fmt.Errorf("collection name not found (no name or __externalId field; set 'name' in the manifest)")
	}
	return collection, nil
}

// resolvedSource returns the spec source with the *.json default pattern
// applied (directories hold one collection per JSON file)
func (spec SmartRestTemplateSpec) resolvedSource() Source {
	source := spec.Source
	if source.Path != "" && len(source.Patterns) == 0 {
		source.Patterns = []string{"*.json"}
	}
	return source
}

// SyncSmartRestTemplates syncs SmartREST 2.0 template collections from
// exported JSON files (one collection per file). Collections are matched via
// the c8y_SmartRest2DeviceIdentifier external identity and only updated when
// they differ from the file (template order does not matter).
func (s *Syncer) SyncSmartRestTemplates(ctx context.Context, specs []SmartRestTemplateSpec) error {
	for index, spec := range specs {
		files, ok := s.resolveSource(SectionSmartRest, fmt.Sprintf("smartrestTemplates[%d]", index), spec.resolvedSource())
		if !ok {
			continue
		}

		if spec.Name != "" && len(files) > 1 {
			s.record(SectionSmartRest, spec.Name, ActionFailed, "",
				fmt.Errorf("source resolved to %d files but 'name' is set; use patterns to select a single file", len(files)))
			continue
		}

		for _, file := range files {
			if s.DryRun {
				item := spec.Name
				if item == "" {
					item = file.Filename
				}
				s.record(SectionSmartRest, item, ActionPlanned, "upsert collection from "+file.Filename, nil)
				continue
			}

			if file.Path == "" {
				s.record(SectionSmartRest, file.Filename, ActionFailed, "",
					fmt.Errorf("collection source must provide a local file (url/linkOnly sources are not supported)"))
				continue
			}

			collection, err := loadSmartRestCollection(file.Path, spec.Name)
			if err != nil {
				s.record(SectionSmartRest, file.Filename, ActionFailed, "parse collection", err)
				continue
			}

			result := s.Client.SmartRestTemplates.Upsert(ctx, smartresttemplates.CreateOptions{
				Name:              collection.Name,
				RequestTemplates:  collection.Templates.RequestTemplates,
				ResponseTemplates: collection.Templates.ResponseTemplates,
				// Annotation: written alongside real changes, but never a
				// reason to update by itself
				Annotations: []model.Fragment{
					model.Frag(SyncToolFragment, syncMeta()),
				},
			})
			if result.Err != nil {
				s.record(SectionSmartRest, collection.Name, ActionFailed, file.Filename, result.Err)
				continue
			}
			s.record(SectionSmartRest, collection.Name, actionFromResult(result.Status, result.Meta), file.Filename, nil)
		}
	}
	return nil
}
