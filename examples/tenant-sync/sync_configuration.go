package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/configuration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// SyncConfiguration uploads configuration files to the configuration repository.
// Each resolved file becomes a configuration item; when the spec sets a name,
// the source must resolve to a single file.
func (s *Syncer) SyncConfiguration(ctx context.Context, specs []ConfigurationSpec) error {
	for index, spec := range specs {
		files, ok := s.resolveSource(SectionConfiguration, fmt.Sprintf("configuration[%d]", index), spec.Source)
		if !ok {
			continue
		}

		if spec.Name != "" && len(files) > 1 {
			s.record(SectionConfiguration, spec.Name, ActionFailed, "",
				fmt.Errorf("source resolved to %d files but 'name' is set; use patterns to select a single file", len(files)))
			continue
		}

		for _, file := range files {
			name := spec.Name
			if name == "" {
				// Derive the name from the filename without extension
				name = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
			}
			item := fmt.Sprintf("%s (%s)", name, spec.ConfigurationType)
			detail := file.Filename
			if file.URL != "" {
				detail = "link → " + file.URL
			}

			if s.DryRun {
				s.record(SectionConfiguration, item, ActionPlanned, detail, nil)
				continue
			}

			createOpts := configuration.CreateOptions{
				Name:              name,
				ConfigurationType: spec.ConfigurationType,
				Description:       spec.Description,
				DeviceType:        spec.DeviceType,
				// Annotation: written alongside real changes, but never a
				// reason to update by itself
				Annotations: []model.Fragment{
					model.Frag(SyncToolFragment, syncMeta()),
				},
			}
			if file.URL != "" {
				createOpts.URL = file.URL
			} else {
				createOpts.File = configuration.UploadFileOptions{
					Name:        file.Filename,
					ContentType: detectContentType(file.Filename),
					FilePath:    file.Path,
				}
			}

			var result op.Result[jsonmodels.Configuration]
			if s.Force {
				result = s.Client.Repository.Configuration.UpsertByName(ctx, createOpts)
			} else {
				result = s.Client.Repository.Configuration.GetOrCreate(ctx, createOpts)
			}
			if result.Err != nil {
				s.record(SectionConfiguration, item, ActionFailed, detail, result.Err)
				continue
			}
			s.record(SectionConfiguration, item, actionFromResult(result.Status, result.Meta), detail, nil)
		}
	}
	return nil
}
