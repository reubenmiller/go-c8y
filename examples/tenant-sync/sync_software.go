package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// SyncSoftware resolves each software source and uploads the packages,
// grouping them by (name, architecture, softwareType) the same way as the
// software-uploader example.
func (s *Syncer) SyncSoftware(ctx context.Context, specs []SoftwareSpec) error {
	for index, spec := range specs {
		files, ok := s.resolveSource(SectionSoftware, fmt.Sprintf("software[%d]", index), spec.Source)
		if !ok {
			continue
		}

		var infos []*SoftwareInfo
		for _, file := range files {
			path := file.Path
			if path == "" {
				// URL-only source: parse from the asset filename
				path = file.Filename
			}

			info, err := ParseSoftwareFromFilename(path, spec.Type, spec.NamePrefix, spec.TypeMappings()...)
			if err != nil {
				s.record(SectionSoftware, file.Filename, ActionFailed, "parse", err)
				continue
			}

			if file.URL != "" {
				info.FilePath = ""
				info.ExternalURL = file.URL
			}

			// Version precedence: manifest override > filename > source hint (release tag)
			if spec.Version != "" {
				info.Version = spec.Version
			} else if info.Version == "" {
				info.Version = file.VersionHint
			}

			if err := ValidateSoftwareInfo(info); err != nil {
				s.record(SectionSoftware, file.Filename, ActionFailed, "validate", err)
				continue
			}
			infos = append(infos, info)
		}

		if len(infos) == 0 {
			continue
		}

		s.syncSoftwareGroup(ctx, infos)
	}
	return nil
}

// syncSoftwareGroup ensures software items exist and uploads versions concurrently
func (s *Syncer) syncSoftwareGroup(ctx context.Context, infos []*SoftwareInfo) {
	groups := GroupBySoftwareNameAndArch(infos)
	softwareIDs := make(map[string]string)

	for key, group := range groups {
		if len(group) == 0 {
			continue
		}
		name := group[0].Name
		arch := group[0].Architecture
		softwareType := group[0].SoftwareType
		item := softwareItemLabel(name, arch, softwareType)

		if s.DryRun {
			softwareIDs[key] = "dry-run"
			s.record(SectionSoftware, item, ActionPlanned, fmt.Sprintf("%d version(s)", len(group)), nil)
			continue
		}

		id, action, err := s.ensureSoftwareItem(ctx, name, arch, softwareType)
		if err != nil {
			s.record(SectionSoftware, item, ActionFailed, "ensure software item", err)
			continue
		}
		softwareIDs[key] = id
		slog.Debug("Ensured software item", "id", id, "name", name, "arch", arch, "type", softwareType, "action", action)
	}

	// Upload versions with a worker pool
	type job struct{ info *SoftwareInfo }
	queue := make(chan job, len(infos))
	for _, info := range infos {
		queue <- job{info}
	}
	close(queue)

	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 20 {
		concurrency = 20
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range queue {
				info := j.info
				key := GetSoftwareKey(info.Name, info.Architecture, info.SoftwareType)
				item := fmt.Sprintf("%s %s", softwareItemLabel(info.Name, info.Architecture, info.SoftwareType), info.Version)

				softwareID, ok := softwareIDs[key]
				if !ok {
					continue // ensure step already recorded the failure
				}

				detail := info.Filename
				if info.ExternalURL != "" {
					detail = "link → " + info.ExternalURL
				}

				if s.DryRun {
					mu.Lock()
					s.record(SectionSoftware, item, ActionPlanned, detail, nil)
					mu.Unlock()
					continue
				}

				action, err := s.uploadSoftwareVersion(ctx, info, softwareID)
				mu.Lock()
				s.record(SectionSoftware, item, action, detail, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func softwareItemLabel(name, arch, softwareType string) string {
	qualifier := softwareType
	if arch != "" {
		qualifier = arch + "/" + softwareType
	}
	return fmt.Sprintf("%s [%s]", name, qualifier)
}

// ensureSoftwareItem creates the software item for a (name, arch, type) group
// if it does not exist, and updates it only when the desired state differs.
// The c8y_TenantSync fragment is passed as an annotation: written alongside
// real changes, but never a reason to update by itself.
func (s *Syncer) ensureSoftwareItem(ctx context.Context, name, arch, softwareType string) (string, Action, error) {
	archAgnostic := arch == "" || arch == "all" || arch == "noarch" || arch == "any"

	query := fmt.Sprintf("name eq '%s' and softwareType eq '%s'", name, softwareType)
	if !archAgnostic {
		query += fmt.Sprintf(" and c8y_Filter.type eq '%s'", arch)
	}

	description := fmt.Sprintf("Software package: %s", name)
	if arch != "" {
		description = fmt.Sprintf("Software package: %s (Architecture: %s)", name, arch)
	}

	body := map[string]any{
		"name":         name,
		"type":         "c8y_Software",
		"softwareType": softwareType,
		"description":  description,
	}
	if !archAgnostic {
		body["c8y_Filter"] = map[string]string{"type": arch}
	}

	result := s.Client.Repository.Software.UpsertWith(ctx, query, body,
		model.Frag(SyncToolFragment, syncMeta()))
	if result.Err != nil {
		return "", ActionFailed, result.Err
	}
	return result.Data.ID(), actionFromResult(result.Status, result.Meta), nil
}

// uploadSoftwareVersion uploads (or links) a single software version
func (s *Syncer) uploadSoftwareVersion(ctx context.Context, info *SoftwareInfo, softwareID string) (Action, error) {
	createOpts := softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    info.Version,
	}
	if info.ExternalURL != "" {
		createOpts.URL = info.ExternalURL
	} else {
		createOpts.File = softwareversions.UploadFileOptions{
			Name:        info.Filename,
			ContentType: detectContentType(info.Filename),
			FilePath:    info.FilePath,
		}
	}

	var result op.Result[jsonmodels.SoftwareVersion]
	if s.Force {
		result = s.Client.Repository.Software.Versions.UpsertByVersion(ctx, createOpts)
	} else {
		result = s.Client.Repository.Software.Versions.GetOrCreateVersion(ctx, createOpts)
	}
	if result.Err != nil {
		return ActionFailed, result.Err
	}
	return actionFromResult(result.Status, result.Meta), nil
}

// detectContentType returns an appropriate content type based on file extension
func detectContentType(filename string) string {
	lower := strings.ToLower(filename)
	doubleExtTypes := map[string]string{
		".pkg.tar.zst": "application/zstd",
		".pkg.tar.xz":  "application/x-xz",
		".tar.gz":      "application/gzip",
		".tar.bz2":     "application/x-bzip2",
		".tar.xz":      "application/x-xz",
		".tar.zst":     "application/zstd",
	}
	for doubleExt, contentType := range doubleExtTypes {
		if strings.HasSuffix(lower, doubleExt) {
			return contentType
		}
	}

	contentTypes := map[string]string{
		".tgz":      "application/gzip",
		".tar":      "application/x-tar",
		".gz":       "application/gzip",
		".bz2":      "application/x-bzip2",
		".xz":       "application/x-xz",
		".zst":      "application/zstd",
		".zip":      "application/zip",
		".7z":       "application/x-7z-compressed",
		".rar":      "application/vnd.rar",
		".bin":      "application/octet-stream",
		".deb":      "application/vnd.debian.binary-package",
		".rpm":      "application/x-rpm",
		".apk":      "application/octet-stream",
		".ipk":      "application/octet-stream",
		".jar":      "application/java-archive",
		".war":      "application/java-archive",
		".ear":      "application/java-archive",
		".exe":      "application/x-msdownload",
		".msi":      "application/x-msi",
		".dmg":      "application/x-apple-diskimage",
		".pkg":      "application/vnd.apple.installer+xml",
		".snap":     "application/vnd.snap",
		".flatpak":  "application/octet-stream",
		".appimage": "application/octet-stream",
	}
	if contentType, ok := contentTypes[strings.ToLower(filepath.Ext(filename))]; ok {
		return contentType
	}
	return "application/octet-stream"
}
