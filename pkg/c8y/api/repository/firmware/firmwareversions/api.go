package firmwareversions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamID = "id"

const ResultProperty = "managedObjects"

const FragmentFirmware = "c8y_Firmware"
const FragmentFirmwareBinary = "c8y_FirmwareBinary"

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:        *s,
		firmware:       firmwareitems.NewService(s),
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
	}
	service.Resolver = NewResolver(service)
	return service
}

// Service api to interact with firmware versions
type Service struct {
	core.Service
	firmware       *firmwareitems.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	Resolver       *Resolver
}

type CreateOptions struct {
	FirmwareID string

	Version string
	URL     string
	File    UploadFileOptions
}

// Create a firmware version under a firmware item
// firmwareID can be a direct ID or use string-based resolver patterns (e.g., "name:MyFirmware")
// Assumes the firmware item exists (does not create it)
func (s *Service) Create(ctx context.Context, firmwareID string, opt CreateOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		// Resolve firmware ID if needed
		resolvedID, err := s.firmware.Resolver.ResolveID(execCtx, firmwareID, nil)
		if err != nil {
			return op.Failed[jsonmodels.FirmwareVersion](
				fmt.Errorf("failed to resolve firmware ID: %w", err),
				false,
			)
		}

		// Upload binary if needed
		url, err := s.uploadBinaryIfNeeded(execCtx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.FirmwareVersion](err, true)
		}

		// Build version body
		versionBody := map[string]any{
			"type": "c8y_FirmwareBinary",
			"c8y_Firmware": map[string]any{
				"version": opt.Version,
			},
		}
		if url != "" {
			versionBody["c8y_Firmware"].(map[string]any)["url"] = url
		}

		return core.Execute(execCtx, s.createB(resolvedID, versionBody), jsonmodels.NewFirmwareVersion)
	}).WithMeta("operation", "create").
		ExecuteOrDefer(ctx)
}

// ResolveID resolves a firmware version identifier to an ID using the resolver
// This is a convenience method that wraps the Resolver.ResolveID
// Supported formats:
//   - "12345" - direct ID
//   - "version:1.0.0:firmware:12345" - lookup by version and firmware ID
//   - "version:1.0.0:name:MyFirmware" - lookup by version and firmware name
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	return s.Resolver.ResolveID(ctx, identifier, meta)
}

// ListOptions filter firmware versions
type ListOptions struct {
	FirmwareID string `url:"-"`
	Version    string `url:"-"`
	Query      string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List firmware versions
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.FirmwareVersion] {
	// Resolve firmware name to ID if needed
	firmwareResult := s.firmware.Get(ctx, opt.FirmwareID, firmwareitems.GetOptions{})
	if firmwareResult.Err != nil {
		return op.Failed[jsonmodels.FirmwareVersion](
			fmt.Errorf("failed to resolve firmware name: %w", firmwareResult.Err),
			true,
		)
	}
	opt.FirmwareID = firmwareResult.Data.ID()

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewFirmwareVersion)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	// Build request directly since managedObjects.listB is now private
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_FirmwareBinary").
			AddFilterEqStr("c8y_Firmware.version", opt.Version).
			AddFilterPart(opt.Query).
			ByGroupID(opt.FirmwareID).
			AddOrderBy("c8y_Firmware.version").
			AddOrderBy("creationTime").
			Build(),
		PaginationOptions: pagination.PaginationOptions{
			CurrentPage: opt.CurrentPage,
			PageSize:    opt.PageSize,
		},
	}
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(listOpts)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// FirmwareVersionIterator provides iteration over firmware versions
type FirmwareVersionIterator = pagination.Iterator[jsonmodels.FirmwareVersion]

// ListAll returns an iterator for all firmware versions
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *FirmwareVersionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.FirmwareVersion] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewFirmwareVersion,
	)
}

type GetOptions struct {
	// Query options
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// DeleteOptions options to delete a firmware version
type DeleteOptions struct {
	// Delete options
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Resolver handles firmware version resolution from various identifier formats
type Resolver struct {
	service *Service
}

// Ref provides helper methods to construct resolver identifier strings
type Ref struct{}

func NewRef() *Ref {
	return &Ref{}
}

// ByID constructs a direct ID reference
func (Ref) ByID(id string) string {
	return id
}

// ByVersion constructs a version-based reference using firmware ID
// Example: Ref{}.ByVersion("1.0.0", "12345") -> "version:1.0.0:firmware:12345"
func (Ref) ByVersion(version, firmwareID string) string {
	return "version:" + version + ":firmware:" + firmwareID
}

// ByVersionAndName constructs a version-based reference using firmware name
// Example: Ref{}.ByVersionAndName("1.0.0", "MyFirmware") -> "version:1.0.0:name:MyFirmware"
func (Ref) ByVersionAndName(version, firmwareName string) string {
	return "version:" + version + ":name:" + firmwareName
}

// NewResolver creates a new firmware version resolver
func NewResolver(service *Service) *Resolver {
	return &Resolver{service: service}
}

// ByID returns the ID directly (for consistency with resolver pattern)
func (r *Resolver) ByID(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("id cannot be empty")
	}
	return id, nil
}

// ByVersion resolves a firmware version by version number and firmware ID
func (r *Resolver) ByVersion(ctx context.Context, version, firmwareID string) (string, error) {
	return r.resolveByVersionAndFirmwareID(ctx, version, firmwareID)
}

// ByVersionAndName resolves a firmware version by version number and firmware name
func (r *Resolver) ByVersionAndName(ctx context.Context, version, firmwareName string) (string, error) {
	return r.resolveByVersionAndFirmwareName(ctx, version, firmwareName)
}

// ResolveID resolves a firmware version identifier string to an ID
// Supported formats:
//   - "12345" - direct ID
//   - "version:1.0.0:firmware:12345" - lookup by version and firmware ID
//   - "version:1.0.0:name:MyFirmware" - lookup by version and firmware name
func (r *Resolver) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	if meta == nil {
		meta = make(map[string]any)
	}

	// Validate identifier is not empty
	if identifier == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}

	// Direct ID (no prefix)
	if !strings.Contains(identifier, ":") {
		meta["resolverType"] = "id"
		return identifier, nil
	}

	parts := strings.Split(identifier, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid identifier format: %s", identifier)
	}

	resolverType := parts[0]

	switch resolverType {
	case "version":
		// Format: "version:1.0.0:firmware:12345" or "version:1.0.0:name:MyFirmware"
		if len(parts) < 4 {
			return "", fmt.Errorf("version resolver requires format 'version:VERSION:firmware:ID' or 'version:VERSION:name:NAME': %s", identifier)
		}
		version := parts[1]
		lookupType := parts[2]
		lookupValue := parts[3]

		meta["resolverType"] = "version"
		meta["version"] = version

		switch lookupType {
		case "firmware":
			// Direct firmware ID
			meta["firmwareID"] = lookupValue
			return r.resolveByVersionAndFirmwareID(ctx, version, lookupValue)

		case "name":
			// Firmware name
			meta["firmwareName"] = lookupValue
			return r.resolveByVersionAndFirmwareName(ctx, version, lookupValue)

		default:
			return "", fmt.Errorf("unsupported version lookup type: %s (must be 'firmware' or 'name')", lookupType)
		}

	default:
		return "", fmt.Errorf("unsupported resolver type: %s (must be 'version')", resolverType)
	}
}

// resolveByVersionAndFirmwareID resolves by version number and firmware ID
func (r *Resolver) resolveByVersionAndFirmwareID(ctx context.Context, version, firmwareID string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if firmwareID == "" {
		return "", fmt.Errorf("firmwareID cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		FirmwareID: firmwareID,
		Version:    version,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup version: %w", listResult.Err)
	}

	// Check if any items were found
	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewFirmwareVersion(item.Bytes())
		return found.ID(), nil
	}

	return "", core.ErrNotFound("version not found: firmware=%s, version=%s", firmwareID, version)
}

// resolveByVersionAndFirmwareName resolves by version number and firmware name
func (r *Resolver) resolveByVersionAndFirmwareName(ctx context.Context, version, firmwareName string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if firmwareName == "" {
		return "", fmt.Errorf("firmware name cannot be empty")
	}

	// First resolve firmware name to ID
	identifier := "name:" + firmwareName

	firmwareResult := r.service.firmware.Get(ctx, identifier, firmwareitems.GetOptions{})
	if firmwareResult.Err != nil {
		return "", fmt.Errorf("failed to resolve firmware name: %w", firmwareResult.Err)
	}

	firmwareID := firmwareResult.Data.ID()
	return r.resolveByVersionAndFirmwareID(ctx, version, firmwareID)
}

// Get retrieves a firmware version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:firmware:12345" - lookup by version and firmware ID
//   - "version:1.0.0:name:MyFirmware" - lookup by version and firmware name
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.FirmwareVersion] {
	// Resolve ID (supports "version:1.0.0:firmware:12345", "version:1.0.0:name:MyFirmware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.FirmwareVersion](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewFirmwareVersion, meta)
}

// Update a firmware version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:firmware:12345" - lookup by version and firmware ID
//   - "version:1.0.0:name:MyFirmware" - lookup by version and firmware name
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.FirmwareVersion] {
	// Resolve ID (supports "version:1.0.0:firmware:12345", "version:1.0.0:name:MyFirmware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.FirmwareVersion](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewFirmwareVersion, meta)
}

// Delete a firmware version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:firmware:12345" - lookup by version and firmware ID
//   - "version:1.0.0:name:MyFirmware" - lookup by version and firmware name
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	// Resolve ID (supports "version:1.0.0:firmware:12345", "version:1.0.0:name:MyFirmware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	meta["id"] = id

	return core.ExecuteNoContent(ctx, s.deleteB(id, opt), meta).IgnoreNotFound()
}

type UploadFileOptions = core.UploadFileOptions

type CreateVersionOptions struct {
	FirmwareName string
	FirmwareID   string

	Version string
	URL     string
	File    UploadFileOptions
}

// CreateVersion creates a firmware version, automatically handling firmware item lookup/creation and binary upload
func (s *Service) CreateVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		// Step 1: Get or create firmware item
		var firmwareID string
		if opt.FirmwareID != "" {
			firmwareID = opt.FirmwareID
		} else if opt.FirmwareName != "" {
			firmwareResult := s.firmware.GetOrCreate(execCtx, firmwareitems.CreateOptions{
				Name: opt.FirmwareName,
			})
			if firmwareResult.Err != nil {
				return op.Failed[jsonmodels.FirmwareVersion](
					fmt.Errorf("failed to get/create firmware item: %w", firmwareResult.Err),
					true,
				)
			}
			firmwareID = firmwareResult.Data.ID()
		} else {
			return op.Failed[jsonmodels.FirmwareVersion](
				fmt.Errorf("must specify FirmwareID or FirmwareName"),
				false,
			)
		}

		// Step 2: Create version (handles binary upload internally)
		createOpt := CreateOptions{
			Version: opt.Version,
			URL:     opt.URL,
			File:    opt.File,
		}
		return s.Create(execCtx, firmwareID, createOpt)
	}).WithMeta("operation", "createVersion").
		ExecuteOrDefer(ctx)
}

// GetOrCreateVersion searches by firmware + version, creating if not found
func (s *Service) GetOrCreateVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		// Define finder function
		finder := func(ctx context.Context) (op.Result[jsonmodels.FirmwareVersion], bool) {
			// First resolve firmware ID
			firmwareID := opt.FirmwareID
			if firmwareID == "" && opt.FirmwareName != "" {
				// Use string-based resolver
				firmwareResult := s.firmware.Get(
					ctx,
					firmwareitems.NewRef().ByName(opt.FirmwareName),
					firmwareitems.GetOptions{},
				)
				if firmwareResult.Err != nil {
					return op.Result[jsonmodels.FirmwareVersion]{}, false
				}
				firmwareID = firmwareResult.Data.ID()
			}

			if firmwareID == "" {
				return op.Result[jsonmodels.FirmwareVersion]{}, false
			}

			// Search for version
			listResult := s.List(ctx, ListOptions{
				FirmwareID: firmwareID,
				Version:    opt.Version,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.FirmwareVersion]{}, false
			}

			// Check if any items were found
			for item := range listResult.Data.Iter() {
				version := jsonmodels.NewFirmwareVersion(item.Bytes())
				result := op.OK(version)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["found"] = true
				result.Meta["lookupMethod"] = "version"
				return result, true
			}

			return op.Result[jsonmodels.FirmwareVersion]{}, false
		}

		// Define creator function
		creator := func(ctx context.Context) op.Result[jsonmodels.FirmwareVersion] {
			createResult := s.CreateVersion(ctx, opt)
			if createResult.Err != nil {
				return createResult
			}
			createResult.Meta["found"] = false
			return createResult
		}

		// Execute get-or-create pattern
		return op.GetOrCreateR(execCtx, finder, creator)
	}).WithMeta("operation", "getOrCreateVersion").
		ExecuteOrDefer(ctx)
}

func (s *Service) DeleteAndCreate(ctx context.Context, versionID string, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		// Step 1: Delete any existing version (if it exists)
		deleteResult := s.Delete(execCtx, versionID, DeleteOptions{})
		if deleteResult.Err != nil {
			return op.Failed[jsonmodels.FirmwareVersion](
				fmt.Errorf("failed to delete existing version: %w", deleteResult.Err),
				true,
			)
		}

		// Step 2: Create new version
		return s.CreateVersion(execCtx, opt)
	}).WithMeta("operation", "deleteAndCreate").
		ExecuteOrDefer(ctx)
}

// Upsert Methods
// These methods follow the finder/updater/creator pattern from op.UpsertR
// They search for an existing version, update it if found (optionally replacing binaries), or create it if not found.
//
// Key behaviors:
// - When updating: deletes old binary (if hosted on tenant) and uploads new one
// - When creating: uploads binary and creates version
// - Returns Status: "Created" or "Updated" with Meta["found"] indicating if resource existed
//
// Available methods:
// - UpsertByVersion: Upserts by version number and firmware ID/name (most common)
// - UpsertWith: Generic query-based upsert (for advanced filtering)

// UpsertByVersion upserts a firmware version by version number and firmware ID
// If the version exists, updates it (optionally replacing the binary).
// If the version doesn't exist, creates it.
// This is useful for ensuring a version exists with the latest binary.
func (s *Service) UpsertByVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		// Resolve firmware ID if needed
		firmwareID := opt.FirmwareID
		if firmwareID == "" && opt.FirmwareName != "" {
			firmwareResult := s.firmware.Get(
				execCtx,
				firmwareitems.NewRef().ByName(opt.FirmwareName),
				firmwareitems.GetOptions{},
			)
			if firmwareResult.Err != nil {
				return op.Failed[jsonmodels.FirmwareVersion](
					fmt.Errorf("failed to resolve firmware name: %w", firmwareResult.Err),
					true,
				)
			}
			firmwareID = firmwareResult.Data.ID()
		}

		if firmwareID == "" {
			return op.Failed[jsonmodels.FirmwareVersion](
				fmt.Errorf("must specify FirmwareID or FirmwareName"),
				false,
			)
		}

		// Build query for lookup
		query := model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_FirmwareBinary").
			AddFilterEqStr("c8y_Firmware.version", opt.Version).
			ByGroupID(firmwareID).
			Build()

		return s.upsertWithQuery(execCtx, query, opt)
	}).WithMeta("operation", "upsertByVersion").
		ExecuteOrDefer(ctx)
}

// UpsertWith provides a generic query-based upsert for firmware versions
// Updates existing version if found, creates if not found
// Example queries:
//   - "c8y_Firmware.version eq '1.0.0'" (requires firmwareID in opt)
func (s *Service) UpsertWith(ctx context.Context, query string, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	return op.Result[jsonmodels.FirmwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		query_ := model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_FirmwareBinary").
			AddFilterPart(query).
			AddOrderBy("c8y_Firmware.version").
			AddOrderBy("creationTime").
			Build()

		return s.upsertWithQuery(execCtx, query_, opt)
	}).WithMeta("operation", "upsertWith").
		ExecuteOrDefer(ctx)
}

// upsertWithQuery is the internal implementation for upsert
func (s *Service) upsertWithQuery(ctx context.Context, query string, opt CreateVersionOptions) op.Result[jsonmodels.FirmwareVersion] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.FirmwareVersion], bool) {
		// Search for existing version
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.FirmwareVersion]{}, false
		}

		// Check if any items were found
		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewFirmwareVersion(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		// Not found
		return op.Result[jsonmodels.FirmwareVersion]{}, false
	}

	// Define updater function
	updater := func(ctx context.Context, existing op.Result[jsonmodels.FirmwareVersion]) op.Result[jsonmodels.FirmwareVersion] {
		// Delete old binary if we're uploading a new one
		if opt.File.FilePath != "" || opt.URL != "" {
			s.deleteBinaryFromURL(ctx, existing.Data.URL())
		}

		// Upload new binary if needed
		url, err := s.uploadBinaryIfNeeded(ctx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.FirmwareVersion](err, true)
		}

		// Build update body
		updateBody := map[string]any{
			"c8y_Firmware": map[string]any{
				"version": opt.Version,
			},
		}
		if url != "" {
			updateBody["c8y_Firmware"].(map[string]any)["url"] = url
		}

		// Update the version
		updateResult := s.Update(ctx, existing.Data.ID(), updateBody)
		if updateResult.Err != nil {
			return updateResult
		}
		return updateResult
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.FirmwareVersion] {
		createResult := s.CreateVersion(ctx, opt)
		if createResult.Err != nil {
			return createResult
		}
		return createResult
	}

	// Execute upsert pattern
	return op.UpsertR(ctx, finder, updater, creator)
}

// extractBinaryID extracts the binary ID from a Cumulocity binary URL
// Returns empty string if the URL is external or invalid
func extractBinaryID(url string) string {
	// Handle relative URLs: /inventory/binaries/{id}
	if strings.HasPrefix(url, "/inventory/binaries/") {
		parts := strings.Split(url, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
	}

	// Handle full URLs: https://tenant.cumulocity.com/inventory/binaries/{id}
	if strings.Contains(url, "/inventory/binaries/") {
		parts := strings.Split(url, "/inventory/binaries/")
		if len(parts) == 2 {
			// Remove any query parameters
			binaryID := strings.Split(parts[1], "?")[0]
			return binaryID
		}
	}

	return ""
}

// uploadBinaryIfNeeded uploads a binary file if needed, or returns the provided URL
// Returns the binary URL and any error encountered
func (s *Service) uploadBinaryIfNeeded(ctx context.Context, binaryUrl string, opt UploadFileOptions) (string, error) {
	// If URL is already provided, use it
	if binaryUrl != "" {
		return binaryUrl, nil
	}

	// Upload the file
	binaryResult := s.binaries.Create(ctx, opt)
	if binaryResult.IsError() {
		return "", fmt.Errorf("failed to upload binary: %w", binaryResult.Err)
	}

	return binaryResult.Data.Self(), nil
}

// deleteBinaryFromURL deletes a binary from a Cumulocity binary URL if it's hosted on the same tenant
// Logs warnings but does not return errors, allowing the operation to continue
func (s *Service) deleteBinaryFromURL(ctx context.Context, url string) {
	if url == "" {
		return
	}

	// Extract binary ID if it's from the same tenant
	binaryID := extractBinaryID(url)
	if binaryID == "" {
		return
	}

	// Delete the binary
	deleteResult := s.binaries.Delete(ctx, binaryID)
	if deleteResult.Err != nil {
		// Log warning but continue - the binary might already be deleted
		slog.Info("failed to delete old binary", "binaryID", binaryID, "err", deleteResult.Err)
	}
}

// Builder methods

func (s *Service) createB(firmwareID string, body any) *core.TryRequest {
	// Build request directly since childAdditions.createB is now private
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("id", firmwareID).
		SetBody(body).
		SetContentType(types.MimeTypeManagedObject).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/inventory/managedObjects/{id}/childAdditions")
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	// Build request directly since managedObjects.getB is now private
	getOpts := managedobjects.GetOptions{
		WithParents: opt.WithParents,
	}
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(getOpts)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	// Build request directly since managedObjects.updateB is now private
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", ID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	// Build request directly since managedObjects.deleteB is now private
	deleteOpts := managedobjects.DeleteOptions{
		ForceCascade: opt.ForceCascade,
	}
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(deleteOpts)).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}
