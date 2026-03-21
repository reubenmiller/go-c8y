package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// UploadConfig contains configuration for the upload process
type UploadConfig struct {
	Client       *api.Client
	Concurrency  int
	SoftwareType string
	DryRun       bool
	Force        bool // Force replacement of existing versions
}

// UploadResult contains the results of an upload operation
type UploadResult struct {
	TotalFiles       int
	SuccessCount     int
	FailureCount     int
	Errors           []UploadError
	SoftwareCreated  int
	SoftwareFound    int
	VersionsCreated  int // Number of versions newly uploaded
	VersionsFound    int // Number of versions that already existed
	VersionsReplaced int // Number of versions that were replaced (force mode)
}

// UploadError represents an error during upload
type UploadError struct {
	FilePath string
	Name     string
	Version  string
	Error    error
}

// ProgressCallback is called to report upload progress
type ProgressCallback func(completed, total int, currentFile string)

// EnsureSoftwarePackages creates or retrieves software packages for all unique names
// Groups by name and architecture to create separate software items per architecture
func EnsureSoftwarePackages(
	ctx context.Context,
	client *api.Client,
	groups map[string][]*SoftwareInfo,
	dryRun bool,
) (map[string]string, []error) {
	softwareIDs := make(map[string]string)
	var errors []error

	for key, infos := range groups {
		if len(infos) == 0 {
			continue
		}

		// Get name, architecture, and software type from first info (all in group have same)
		name := infos[0].Name
		arch := infos[0].Architecture
		softwareType := infos[0].SoftwareType

		slog.Debug("Processing software package",
			"name", name,
			"type", softwareType,
			"architecture", arch,
			"version_count", len(infos))

		if dryRun {
			softwareIDs[key] = fmt.Sprintf("dry-run-id-%s", key)
			continue
		}

		// Build description with architecture info
		description := fmt.Sprintf("Software package: %s", name)
		if arch != "" {
			description = fmt.Sprintf("Software package: %s (Architecture: %s)", name, arch)
		}

		// Build query to find software by name, type, and deviceType (architecture)
		var query string
		if arch != "" {
			// Include deviceType in query to find software with matching architecture
			query = fmt.Sprintf("name eq '%s' and softwareType eq '%s' and c8y_Filter.type eq '%s'", name, softwareType, arch)
			slog.Debug("Looking up software with architecture",
				"name", name,
				"type", softwareType,
				"architecture", arch,
				"query", query)
		} else {
			// No architecture - just query by name and type
			query = fmt.Sprintf("name eq '%s' and softwareType eq '%s'", name, softwareType)
			slog.Debug("Looking up software without architecture",
				"name", name,
				"type", softwareType,
				"query", query)
		}

		// Create body
		body := map[string]any{
			"name":         name,
			"type":         "c8y_Software",
			"softwareType": softwareType,
			"description":  description,
			// Add fragment to identify software uploaded by this tool
			"c8y_SoftwareUploader": map[string]any{
				"uploadedAt": time.Now(),
				"tool":       "software-uploader",
			},
		}

		// store architecture and use it as the device type filter
		if arch != "" {
			body["arch"] = arch
			body["c8y_Filter"] = map[string]string{
				"type": arch,
			}
		}

		// Use UpsertWith to ensure metadata stays up-to-date
		result := client.Repository.Software.UpsertWith(
			ctx,
			query,
			body,
		)

		if result.Err != nil {
			slog.Error("Failed to create/find software",
				"name", name,
				"type", softwareType,
				"architecture", arch,
				"error", result.Err)
			errors = append(errors, fmt.Errorf("failed to create/get software %s: %w", name, result.Err))
			continue
		}

		softwareID := result.Data.ID()

		// Check if this was newly created, updated, or already existed
		if result.Status == "Created" || (result.Meta != nil && result.Meta["found"] == false) {
			slog.Info("Created new software item",
				"id", softwareID,
				"name", name,
				"type", softwareType,
				"architecture", arch)
		} else if result.Status == "Updated" {
			slog.Info("Updated software item",
				"id", softwareID,
				"name", name,
				"type", softwareType,
				"architecture", arch)
		} else {
			slog.Debug("Found existing software item (no changes)",
				"id", softwareID,
				"name", name,
				"type", softwareType,
				"architecture", arch)
		}

		softwareIDs[key] = softwareID
	}

	return softwareIDs, errors
}

// UploadSoftwareVersions uploads software versions concurrently
func UploadSoftwareVersions(
	ctx context.Context,
	config *UploadConfig,
	infos []*SoftwareInfo,
	progressCallback ProgressCallback,
) *UploadResult {
	result := &UploadResult{
		TotalFiles: len(infos),
	}

	if len(infos) == 0 {
		return result
	}

	// Group by software name and architecture
	groups := GroupBySoftwareNameAndArch(infos)

	// Ensure all software packages exist
	if progressCallback != nil {
		progressCallback(0, len(infos), "Creating software packages...")
	}

	softwareIDs, ensureErrors := EnsureSoftwarePackages(
		ctx,
		config.Client,
		groups,
		config.DryRun,
	)

	if len(ensureErrors) > 0 {
		for _, err := range ensureErrors {
			result.Errors = append(result.Errors, UploadError{
				Error: err,
			})
			result.FailureCount++
		}
		return result
	}

	result.SoftwareCreated = len(softwareIDs)

	// Create a work queue
	workQueue := make(chan *SoftwareInfo, len(infos))
	for _, info := range infos {
		workQueue <- info
	}
	close(workQueue)

	// Track progress
	var completed atomic.Int32
	var successCount atomic.Int32
	var versionsCreated atomic.Int32
	var versionsFound atomic.Int32
	var versionsReplaced atomic.Int32
	var mu sync.Mutex
	var uploadErrors []UploadError

	// Worker pool
	var wg sync.WaitGroup
	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 20 {
		concurrency = 20
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for info := range workQueue {
				// Get software ID using name+arch+type key
				key := GetSoftwareKey(info.Name, info.Architecture, info.SoftwareType)
				softwareID, ok := softwareIDs[key]
				if !ok {
					mu.Lock()
					uploadErrors = append(uploadErrors, UploadError{
						FilePath: info.FilePath,
						Name:     info.Name,
						Version:  info.Version,
						Error:    fmt.Errorf("software package not found for %s", key),
					})
					mu.Unlock()
					completed.Add(1)
					if progressCallback != nil {
						progressCallback(int(completed.Load()), len(infos), info.Filename)
					}
					continue
				}

				// Upload version
				status, err := uploadVersion(ctx, config, info, softwareID)

				completed.Add(1)

				if err != nil {
					mu.Lock()
					uploadErrors = append(uploadErrors, UploadError{
						FilePath: info.FilePath,
						Name:     info.Name,
						Version:  info.Version,
						Error:    err,
					})
					mu.Unlock()
				} else {
					successCount.Add(1)
					if status.Created {
						versionsCreated.Add(1)
					} else if status.Replaced {
						versionsReplaced.Add(1)
					} else if status.Found {
						versionsFound.Add(1)
					}
				}

				if progressCallback != nil {
					progressCallback(int(completed.Load()), len(infos), info.Filename)
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	result.SuccessCount = int(successCount.Load())
	result.FailureCount = len(uploadErrors)
	result.Errors = uploadErrors
	result.VersionsCreated = int(versionsCreated.Load())
	result.VersionsFound = int(versionsFound.Load())
	result.VersionsReplaced = int(versionsReplaced.Load())

	return result
}

// VersionStatus indicates what happened during version upload
type VersionStatus struct {
	Created  bool // Newly created version
	Replaced bool // Existing version was replaced (force mode)
	Found    bool // Version already existed and was not modified
}

// uploadVersion uploads a single software version
// Returns (status VersionStatus, error)
func uploadVersion(
	ctx context.Context,
	config *UploadConfig,
	info *SoftwareInfo,
	softwareID string,
) (VersionStatus, error) {
	if config.DryRun {
		return VersionStatus{}, nil
	}

	slog.Debug("Uploading version",
		"software_id", softwareID,
		"version", info.Version,
		"file", info.Filename,
		"force", config.Force)

	createOpts := softwareversions.CreateVersionOptions{
		SoftwareID: softwareID,
		Version:    info.Version,
		File: softwareversions.UploadFileOptions{
			Name:        info.Filename,
			ContentType: detectContentType(info.Filename),
			FilePath:    info.FilePath,
		},
	}

	// Choose method based on Force mode:
	// - Force mode: UpsertByVersion (always updates if found, replacing binary)
	// - Normal mode: GetOrCreateVersion (skips if already exists)
	var result op.Result[jsonmodels.SoftwareVersion]
	if config.Force {
		result = config.Client.Repository.Software.Versions.UpsertByVersion(ctx, createOpts)
	} else {
		result = config.Client.Repository.Software.Versions.GetOrCreateVersion(ctx, createOpts)
	}

	if result.Err != nil {
		slog.Error("Failed to upload version",
			"software_id", softwareID,
			"version", info.Version,
			"file", info.Filename,
			"error", result.Err)
		return VersionStatus{}, fmt.Errorf("upload failed: %w", result.Err)
	}

	versionID := result.Data.ID()

	// Parse result status
	// Meta["found"]: true if version existed, false if newly created
	// Status: "Created" for new, "Updated" for replaced
	var created bool
	var replaced bool
	var found bool

	if result.Meta != nil {
		if foundVal, ok := result.Meta["found"].(bool); ok {
			found = foundVal
			created = !foundVal && result.Status == "Created"
			replaced = foundVal && result.Status == "Updated"
		}
	}

	// Fallback to checking Status if Meta["found"] is not set
	if result.Meta == nil || result.Meta["found"] == nil {
		created = result.Status == "Created"
		replaced = result.Status == "Updated"
		found = !created && !replaced
	}

	// Log the outcome
	if created {
		slog.Info("Uploaded new software version",
			"software_id", softwareID,
			"version_id", versionID,
			"version", info.Version,
			"file", info.Filename)
	} else if replaced {
		slog.Info("Replaced existing software version",
			"software_id", softwareID,
			"version_id", versionID,
			"version", info.Version,
			"file", info.Filename)
	} else {
		slog.Debug("Version already exists",
			"software_id", softwareID,
			"version_id", versionID,
			"version", info.Version,
			"file", info.Filename)
	}

	return VersionStatus{Created: created, Replaced: replaced, Found: found}, nil
}

// detectContentType returns an appropriate content type based on file extension
func detectContentType(filename string) string {
	ext := filepath.Ext(filename)

	// Check for triple/double extensions first (before filepath.Ext strips only the last part)
	lower := strings.ToLower(filename)
	doubleExtTypes := map[string]string{
		".pkg.tar.zst": "application/zstd",
		".pkg.tar.xz":  "application/x-xz",
		".tar.gz":      "application/gzip",
		".tar.bz2":     "application/x-bzip2",
		".tar.xz":      "application/x-xz",
		".tar.zst":     "application/zstd",
	}
	for doubleExt, ct := range doubleExtTypes {
		if strings.HasSuffix(lower, doubleExt) {
			return ct
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

	if ct, ok := contentTypes[strings.ToLower(ext)]; ok {
		return ct
	}

	return "application/octet-stream"
}
