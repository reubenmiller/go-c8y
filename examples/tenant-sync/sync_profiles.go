package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/configuration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
)

// SyncDeviceProfiles creates/updates device profiles (type c8y_Profile)
// referencing firmware, software and configuration items in the tenant.
// References are resolved to their binary URLs, which is why this section
// runs after the repository sections.
func (s *Syncer) SyncDeviceProfiles(ctx context.Context, specs []DeviceProfileSpec) error {
	for _, spec := range specs {
		if s.DryRun {
			s.record(SectionDeviceProfiles, spec.Name, ActionPlanned, "", nil)
			continue
		}

		body, err := s.buildProfileBody(ctx, spec)
		if err != nil {
			s.record(SectionDeviceProfiles, spec.Name, ActionFailed, "resolve references", err)
			continue
		}

		query := fmt.Sprintf("type eq 'c8y_Profile' and name eq '%s'", escapeQueryValue(spec.Name))
		result := s.Client.ManagedObjects.GetOrCreateWith(ctx, body, query)
		if result.Err != nil {
			s.record(SectionDeviceProfiles, spec.Name, ActionFailed, "", result.Err)
			continue
		}

		action := actionFromResult(result.Status, result.Meta)
		if action != ActionCreated {
			// Profile already existed: update it only when it differs from the
			// manifest so re-applying an unchanged manifest is a no-op.
			// The c8y_TenantSync annotation is deliberately not compared.
			upToDate := jsonEqual(body["c8y_DeviceProfile"], result.Data.Get("c8y_DeviceProfile").Raw) &&
				(spec.DeviceType == "" || result.Data.Get("c8y_Filter.type").String() == spec.DeviceType)
			if upToDate {
				action = ActionUnchanged
			} else {
				updateResult := s.Client.ManagedObjects.Update(ctx, result.Data.ID(), body)
				if updateResult.Err != nil {
					s.record(SectionDeviceProfiles, spec.Name, ActionFailed, "update", updateResult.Err)
					continue
				}
				action = ActionUpdated
			}
		}
		s.record(SectionDeviceProfiles, spec.Name, action, "", nil)
	}
	return nil
}

func escapeQueryValue(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func (s *Syncer) buildProfileBody(ctx context.Context, spec DeviceProfileSpec) (map[string]any, error) {
	profile := map[string]any{}

	if spec.Firmware != nil {
		url, err := s.lookupFirmwareVersionURL(ctx, spec.Firmware.Name, spec.Firmware.Version)
		if err != nil {
			return nil, fmt.Errorf("firmware %s/%s: %w", spec.Firmware.Name, spec.Firmware.Version, err)
		}
		profile["firmware"] = map[string]any{
			"name":    spec.Firmware.Name,
			"version": spec.Firmware.Version,
			"url":     url,
		}
	}

	if len(spec.Software) > 0 {
		software := make([]map[string]any, 0, len(spec.Software))
		for _, ref := range spec.Software {
			url, err := s.lookupSoftwareVersionURL(ctx, ref.Name, ref.Version)
			if err != nil {
				return nil, fmt.Errorf("software %s/%s: %w", ref.Name, ref.Version, err)
			}
			action := ref.Action
			if action == "" {
				action = "install"
			}
			software = append(software, map[string]any{
				"name":    ref.Name,
				"version": ref.Version,
				"url":     url,
				"action":  action,
			})
		}
		profile["software"] = software
	}

	if len(spec.Configuration) > 0 {
		configs := make([]map[string]any, 0, len(spec.Configuration))
		for _, ref := range spec.Configuration {
			url, err := s.lookupConfigurationURL(ctx, ref.Name, ref.Type)
			if err != nil {
				return nil, fmt.Errorf("configuration %s: %w", ref.Name, err)
			}
			configs = append(configs, map[string]any{
				"name": ref.Name,
				"type": ref.Type,
				"url":  url,
			})
		}
		profile["configuration"] = configs
	}

	body := map[string]any{
		"name":              spec.Name,
		"type":              "c8y_Profile",
		"c8y_DeviceProfile": profile,
		SyncToolFragment:    syncMeta(),
	}
	if spec.DeviceType != "" {
		body["c8y_Filter"] = map[string]any{"type": spec.DeviceType}
	}
	return body, nil
}

func (s *Syncer) lookupFirmwareVersionURL(ctx context.Context, name, version string) (string, error) {
	firmware := s.Client.Repository.Firmware.Get(ctx, firmwareitems.NewRef().ByName(name), firmwareitems.GetOptions{})
	if firmware.Err != nil {
		return "", fmt.Errorf("firmware item not found: %w", firmware.Err)
	}

	versions := s.Client.Repository.Firmware.Versions.List(ctx, firmwareversions.ListOptions{
		FirmwareID: firmware.Data.ID(),
		Version:    version,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})
	if versions.Err != nil {
		return "", versions.Err
	}
	for item := range versions.Data.Iter() {
		return item.Get("c8y_Firmware.url").String(), nil
	}
	return "", fmt.Errorf("version not found")
}

func (s *Syncer) lookupSoftwareVersionURL(ctx context.Context, name, version string) (string, error) {
	software := s.Client.Repository.Software.Get(ctx, softwareitems.NewRef().ByName(name), softwareitems.GetOptions{})
	if software.Err != nil {
		return "", fmt.Errorf("software item not found: %w", software.Err)
	}

	versions := s.Client.Repository.Software.Versions.List(ctx, softwareversions.ListOptions{
		SoftwareID: software.Data.ID(),
		Version:    version,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})
	if versions.Err != nil {
		return "", versions.Err
	}
	for item := range versions.Data.Iter() {
		return item.Get("c8y_Software.url").String(), nil
	}
	return "", fmt.Errorf("version not found")
}

func (s *Syncer) lookupConfigurationURL(ctx context.Context, name, configurationType string) (string, error) {
	result := s.Client.Repository.Configuration.List(ctx, configuration.ListOptions{
		Name:              name,
		ConfigurationType: configurationType,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})
	if result.Err != nil {
		return "", result.Err
	}
	for item := range result.Data.Iter() {
		return item.Get("url").String(), nil
	}
	return "", fmt.Errorf("configuration item not found")
}
