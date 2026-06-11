package main

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// SyncFirmware resolves each firmware source and uploads the images as
// firmware versions, creating the firmware items as needed.
func (s *Syncer) SyncFirmware(ctx context.Context, specs []FirmwareSpec) error {
	for index, spec := range specs {
		files, ok := s.resolveSource(SectionFirmware, fmt.Sprintf("firmware[%d]", index), spec.Source)
		if !ok {
			continue
		}

		// Parse each file and group by firmware name
		groups := make(map[string][]*FirmwareInfo)
		for _, file := range files {
			info, err := ParseFirmwareFromFilename(file.Path, file.Filename, spec.VersionPattern)
			if err != nil {
				s.record(SectionFirmware, file.Filename, ActionFailed, "parse", err)
				continue
			}
			info.URL = file.URL

			if spec.Name != "" {
				info.Name = spec.Name
			}
			// Version precedence: manifest override > filename > source hint (release tag)
			if spec.Version != "" {
				info.Version = spec.Version
			} else if info.Version == "" {
				info.Version = file.VersionHint
			}

			if info.Name == "" {
				s.record(SectionFirmware, file.Filename, ActionFailed, "",
					fmt.Errorf("could not determine firmware name from filename (set 'name' in the manifest)"))
				continue
			}
			if info.Version == "" {
				s.record(SectionFirmware, file.Filename, ActionFailed, "",
					fmt.Errorf("could not determine version from filename (set 'version' or 'versionPattern' in the manifest)"))
				continue
			}

			// deviceType may contain placeholders derived from the artifact,
			// e.g. "linux-{name}"
			info.DeviceType = info.ExpandPlaceholders(spec.DeviceType)

			groups[info.Name] = append(groups[info.Name], info)
		}

		for name, infos := range groups {
			s.syncFirmwareItem(ctx, spec, name, infos)
		}
	}
	return nil
}

func (s *Syncer) syncFirmwareItem(ctx context.Context, spec FirmwareSpec, name string, infos []*FirmwareInfo) {
	if s.DryRun {
		detail := fmt.Sprintf("%d version(s)", len(infos))
		if infos[0].DeviceType != "" {
			detail += fmt.Sprintf(", deviceType: %s", infos[0].DeviceType)
		}
		s.record(SectionFirmware, name, ActionPlanned, detail, nil)
		for _, info := range infos {
			s.record(SectionFirmware, fmt.Sprintf("%s %s", name, info.Version), ActionPlanned, firmwareDetail(info), nil)
		}
		return
	}

	description := spec.Description
	if description == "" {
		description = fmt.Sprintf("Firmware: %s", name)
	}

	// The deviceType is expanded per artifact; the firmware item carries the
	// one from the first artifact of the group (placeholder templates should
	// only use {name} so all versions of an item agree)
	deviceType := infos[0].DeviceType

	// The c8y_TenantSync fragment is an annotation: written alongside real
	// changes, but never a reason to update by itself
	itemResult := s.Client.Repository.Firmware.UpsertByName(ctx, firmwareitems.CreateOptions{
		Name:        name,
		Description: description,
		DeviceType:  deviceType,
		Annotations: []model.Fragment{
			model.Frag(SyncToolFragment, syncMeta()),
		},
	})
	if itemResult.Err != nil {
		s.record(SectionFirmware, name, ActionFailed, "ensure firmware item", itemResult.Err)
		return
	}
	firmwareID := itemResult.Data.ID()
	s.record(SectionFirmware, name, actionFromResult(itemResult.Status, itemResult.Meta), "", nil)

	for _, info := range infos {
		item := fmt.Sprintf("%s %s", name, info.Version)

		createOpts := firmwareversions.CreateVersionOptions{
			FirmwareID: firmwareID,
			Version:    info.Version,
		}
		if info.URL != "" {
			createOpts.URL = info.URL
		} else {
			createOpts.File = firmwareversions.UploadFileOptions{
				Name:        info.Filename,
				ContentType: detectContentType(info.Filename),
				FilePath:    info.FilePath,
			}
		}

		var result op.Result[jsonmodels.FirmwareVersion]
		if s.Force {
			result = s.Client.Repository.Firmware.Versions.UpsertByVersion(ctx, createOpts)
		} else {
			result = s.Client.Repository.Firmware.Versions.GetOrCreateVersion(ctx, createOpts)
		}
		if result.Err != nil {
			s.record(SectionFirmware, item, ActionFailed, firmwareDetail(info), result.Err)
			continue
		}
		s.record(SectionFirmware, item, actionFromResult(result.Status, result.Meta), firmwareDetail(info), nil)
	}
}

// firmwareDetail describes where a firmware version comes from: an uploaded
// file or an external link
func firmwareDetail(info *FirmwareInfo) string {
	if info.URL != "" {
		return "link → " + info.URL
	}
	return info.Filename
}
