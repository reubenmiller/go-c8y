package softwareversions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/binaries"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamId = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:        *s,
		software:       softwareitems.NewService(s),
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
	}
	service.Resolver = NewResolver(service)
	return service
}

// Service api to interact with software versions
type Service struct {
	core.Service
	software       *softwareitems.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	Resolver       *Resolver
}

type CreateOptions struct {
	SoftwareID string

	Version string
	URL     string
	File    UploadFileOptions
}

// Create a software version under a software item
// softwareID can be a direct ID or use string-based resolver patterns (e.g., "name:MySoftware")
// Assumes the software item exists (does not create it)
func (s *Service) Create(ctx context.Context, softwareID string, opt CreateOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		// Resolve software ID if needed
		resolvedID, err := s.software.Resolver.ResolveID(execCtx, softwareID, nil)
		if err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("failed to resolve software ID: %w", err),
				false,
			)
		}

		// Upload binary if needed
		url, err := s.uploadBinaryIfNeeded(execCtx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](err, true)
		}

		// Build version body
		versionBody := map[string]any{
			"type": "c8y_SoftwareBinary",
			"c8y_Software": map[string]any{
				"version": opt.Version,
			},
		}
		if url != "" {
			versionBody["c8y_Software"].(map[string]any)["url"] = url
		}

		return core.Execute(execCtx, s.createB(resolvedID, versionBody), jsonmodels.NewSoftwareVersion)
	}).WithMeta("operation", "create").
		ExecuteOrDefer(ctx)
}

// ResolveID resolves a software version identifier to an ID using the resolver
// This is a convenience method that wraps the Resolver.ResolveID
// Supported formats:
//   - "12345" - direct ID
//   - "version:1.0.0:software:12345" - lookup by version and software ID
//   - "version:1.0.0:name:MySoftware" - lookup by version and software name
//   - "version:1.0.0:name:MySoftware:application" - lookup by version, software name and type
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	return s.Resolver.ResolveID(ctx, identifier, meta)
}

// ListOptions filter software versions
type ListOptions struct {
	SoftwareID string `url:"-"`
	Version    string `url:"-"`
	Query      string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List software versions
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Resolve software name to ID if needed
	softwareResult := s.software.Get(ctx, opt.SoftwareID, softwareitems.GetOptions{})
	if softwareResult.Err != nil {
		return op.Failed[jsonmodels.SoftwareVersion](
			fmt.Errorf("failed to resolve software name: %w", softwareResult.Err),
			true,
		)
	}
	opt.SoftwareID = softwareResult.Data.ID()

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSoftwareVersion)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	// Build request directly since managedObjects.listB is now private
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_SoftwareBinary").
			AddFilterEqStr("c8y_Software.version", opt.Version).
			AddFilterPart(opt.Query).
			ByGroupID(opt.SoftwareID).
			AddOrderBy("c8y_Software.version").
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

// SoftwareVersionIterator provides iteration over software versions
type SoftwareVersionIterator = pagination.Iterator[jsonmodels.SoftwareVersion]

// ListAll returns an iterator for all software versions
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SoftwareVersionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.SoftwareVersion] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewSoftwareVersion,
	)
}

type GetOptions struct {
	// Query options
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// DeleteOptions options to delete a software version
type DeleteOptions struct {
	// Delete options
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Resolver handles software version resolution from various identifier formats
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

// ByVersion constructs a version-based reference using software ID
// Example: Ref{}.ByVersion("1.0.0", "12345") -> "version:1.0.0:software:12345"
func (Ref) ByVersion(version, softwareID string) string {
	return "version:" + version + ":software:" + softwareID
}

// ByVersionAndName constructs a version-based reference using software name with optional type
// Examples:
//   - Ref{}.ByVersionAndName("1.0.0", "MySoftware") -> "version:1.0.0:name:MySoftware"
//   - Ref{}.ByVersionAndName("1.0.0", "MySoftware", "application") -> "version:1.0.0:name:MySoftware:application"
func (Ref) ByVersionAndName(version, softwareName string, softwareType ...string) string {
	if len(softwareType) > 0 && softwareType[0] != "" {
		return "version:" + version + ":name:" + softwareName + ":" + softwareType[0]
	}
	return "version:" + version + ":name:" + softwareName
}

// NewResolver creates a new software version resolver
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

// ByVersion resolves a software version by version number and software ID
func (r *Resolver) ByVersion(ctx context.Context, version, softwareID string) (string, error) {
	return r.resolveByVersionAndSoftwareID(ctx, version, softwareID)
}

// ByVersionAndName resolves a software version by version number and software name (optionally with type)
func (r *Resolver) ByVersionAndName(ctx context.Context, version, softwareName, softwareType string) (string, error) {
	return r.resolveByVersionAndSoftwareName(ctx, version, softwareName, softwareType)
}

// ResolveID resolves a software version identifier string to an ID
// Supported formats:
//   - "12345" - direct ID
//   - "version:1.0.0:software:12345" - lookup by version and software ID
//   - "version:1.0.0:name:MySoftware" - lookup by version and software name
//   - "version:1.0.0:name:MySoftware:application" - lookup by version, software name and type
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
		// Format: "version:1.0.0:software:12345" or "version:1.0.0:name:MySoftware" or "version:1.0.0:name:MySoftware:application"
		if len(parts) < 4 {
			return "", fmt.Errorf("version resolver requires format 'version:VERSION:software:ID' or 'version:VERSION:name:NAME[:TYPE]': %s", identifier)
		}
		version := parts[1]
		lookupType := parts[2]
		lookupValue := parts[3]

		meta["resolverType"] = "version"
		meta["version"] = version

		switch lookupType {
		case "software":
			// Direct software ID
			meta["softwareID"] = lookupValue
			return r.resolveByVersionAndSoftwareID(ctx, version, lookupValue)

		case "name":
			// Software name (optionally with type)
			// Check if there's a 5th part for software type
			softwareType := ""
			if len(parts) == 5 {
				softwareType = parts[4]
			}
			meta["softwareName"] = lookupValue
			if softwareType != "" {
				meta["softwareType"] = softwareType
			}
			return r.resolveByVersionAndSoftwareName(ctx, version, lookupValue, softwareType)

		default:
			return "", fmt.Errorf("unsupported version lookup type: %s (must be 'software' or 'name')", lookupType)
		}

	default:
		return "", fmt.Errorf("unsupported resolver type: %s (must be 'version')", resolverType)
	}
}

// resolveByVersionAndSoftwareID resolves by version number and software ID
func (r *Resolver) resolveByVersionAndSoftwareID(ctx context.Context, version, softwareID string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if softwareID == "" {
		return "", fmt.Errorf("softwareID cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		SoftwareID: softwareID,
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
		found := jsonmodels.NewSoftwareVersion(item.Bytes())
		return found.ID(), nil
	}

	return "", fmt.Errorf("version not found: software=%s, version=%s", softwareID, version)
}

// resolveByVersionAndSoftwareName resolves by version number and software name (optionally with type)
func (r *Resolver) resolveByVersionAndSoftwareName(ctx context.Context, version, softwareName, softwareType string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if softwareName == "" {
		return "", fmt.Errorf("software name cannot be empty")
	}

	// First resolve software name to ID
	identifier := "name:" + softwareName
	if softwareType != "" {
		identifier = "name:" + softwareName + ":" + softwareType
	}

	softwareResult := r.service.software.Get(ctx, identifier, softwareitems.GetOptions{})
	if softwareResult.Err != nil {
		return "", fmt.Errorf("failed to resolve software name: %w", softwareResult.Err)
	}

	softwareID := softwareResult.Data.ID()
	return r.resolveByVersionAndSoftwareID(ctx, version, softwareID)
}

// Get retrieves a software version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:software:12345" - lookup by version and software ID
//   - "version:1.0.0:name:MySoftware" - lookup by version and software name
//   - "version:1.0.0:name:MySoftware:application" - lookup by version, software name and type
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Resolve ID (supports "version:1.0.0:software:12345", "version:1.0.0:name:MySoftware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.SoftwareVersion](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewSoftwareVersion, meta)
}

// Update a software version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:software:12345" - lookup by version and software ID
//   - "version:1.0.0:name:MySoftware" - lookup by version and software name
//   - "version:1.0.0:name:MySoftware:application" - lookup by version, software name and type
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.SoftwareVersion] {
	// Resolve ID (supports "version:1.0.0:software:12345", "version:1.0.0:name:MySoftware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.SoftwareVersion](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewSoftwareVersion, meta)
}

// Delete a software version
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "version:1.0.0:software:12345" - lookup by version and software ID
//   - "version:1.0.0:name:MySoftware" - lookup by version and software name
//   - "version:1.0.0:name:MySoftware:application" - lookup by version, software name and type
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	// Resolve ID (supports "version:1.0.0:software:12345", "version:1.0.0:name:MySoftware", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	meta["id"] = id

	return core.ExecuteNoContent(ctx, s.deleteB(id, opt), meta).IgnoreNotFound()
}

type UploadFileOptions = core.UploadFileOptions

type CreateVersionOptions struct {
	SoftwareName string
	SoftwareType string
	SoftwareID   string

	Version string
	URL     string
	File    UploadFileOptions
}

// CreateVersion creates a software version, automatically handling software item lookup/creation and binary upload
func (s *Service) CreateVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		// Step 1: Get or create software item
		var softwareID string
		if opt.SoftwareID != "" {
			softwareID = opt.SoftwareID
		} else if opt.SoftwareName != "" {
			softwareResult := s.software.GetOrCreateByName(execCtx, opt.SoftwareName, opt.SoftwareType, map[string]any{
				"name":         opt.SoftwareName,
				"type":         "c8y_Software",
				"softwareType": opt.SoftwareType,
			})
			if softwareResult.Err != nil {
				return op.Failed[jsonmodels.SoftwareVersion](
					fmt.Errorf("failed to get/create software item: %w", softwareResult.Err),
					true,
				)
			}
			softwareID = softwareResult.Data.ID()
		} else {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("must specify SoftwareID or SoftwareName"),
				false,
			)
		}

		// Step 2: Create version (handles binary upload internally)
		createOpt := CreateOptions{
			Version: opt.Version,
			URL:     opt.URL,
			File:    opt.File,
		}
		return s.Create(execCtx, softwareID, createOpt)
	}).WithMeta("operation", "createVersion").
		ExecuteOrDefer(ctx)
}

// GetOrCreateVersion searches by software + version, creating if not found
func (s *Service) GetOrCreateVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		// Define finder function
		finder := func(ctx context.Context) (op.Result[jsonmodels.SoftwareVersion], bool) {
			// First resolve software ID
			softwareID := opt.SoftwareID
			if softwareID == "" && opt.SoftwareName != "" {
				// Use string-based resolver
				softwareResult := s.software.Get(
					ctx,
					softwareitems.NewRef().ByName(opt.SoftwareName, opt.SoftwareType),
					softwareitems.GetOptions{},
				)
				if softwareResult.Err != nil {
					return op.Result[jsonmodels.SoftwareVersion]{}, false
				}
				softwareID = softwareResult.Data.ID()
			}

			if softwareID == "" {
				return op.Result[jsonmodels.SoftwareVersion]{}, false
			}

			// Search for version
			listResult := s.List(ctx, ListOptions{
				SoftwareID: softwareID,
				Version:    opt.Version,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.SoftwareVersion]{}, false
			}

			// Check if any items were found
			for item := range listResult.Data.Iter() {
				version := jsonmodels.NewSoftwareVersion(item.Bytes())
				result := op.OK(version)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["found"] = true
				result.Meta["lookupMethod"] = "version"
				return result, true
			}

			return op.Result[jsonmodels.SoftwareVersion]{}, false
		}

		// Define creator function
		creator := func(ctx context.Context) op.Result[jsonmodels.SoftwareVersion] {
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

func (s *Service) DeleteAndCreate(ctx context.Context, versionID string, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		// Step 1: Delete any existing version (if it exists)
		deleteResult := s.Delete(execCtx, versionID, DeleteOptions{})
		if deleteResult.Err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](
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
// - UpsertByVersion: Upserts by version number and software ID/name (most common)
// - UpsertWith: Generic query-based upsert (for advanced filtering)

// UpsertByVersion upserts a software version by version number and software ID
// If the version exists, updates it (optionally replacing the binary).
// If the version doesn't exist, creates it.
// This is useful for ensuring a version exists with the latest binary.
func (s *Service) UpsertByVersion(ctx context.Context, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		// Resolve software ID if needed
		softwareID := opt.SoftwareID
		if softwareID == "" && opt.SoftwareName != "" {
			softwareResult := s.software.Get(
				execCtx,
				softwareitems.NewRef().ByName(opt.SoftwareName, opt.SoftwareType),
				softwareitems.GetOptions{},
			)
			if softwareResult.Err != nil {
				return op.Failed[jsonmodels.SoftwareVersion](
					fmt.Errorf("failed to resolve software name: %w", softwareResult.Err),
					true,
				)
			}
			softwareID = softwareResult.Data.ID()
		}

		if softwareID == "" {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("must specify SoftwareID or SoftwareName"),
				false,
			)
		}

		// Build query for lookup
		query := model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_SoftwareBinary").
			AddFilterEqStr("c8y_Software.version", opt.Version).
			ByGroupID(softwareID).
			Build()

		return s.upsertWithQuery(execCtx, query, opt)
	}).WithMeta("operation", "upsertByVersion").
		ExecuteOrDefer(ctx)
}

// UpsertWith provides a generic query-based upsert for software versions
// Updates existing version if found, creates if not found
// Example queries:
//   - "c8y_Software.version eq '1.0.0'" (requires softwareID in opt)
func (s *Service) UpsertWith(ctx context.Context, query string, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	return op.Result[jsonmodels.SoftwareVersion]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		query_ := model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_SoftwareBinary").
			AddFilterPart(query).
			AddOrderBy("c8y_Software.version").
			AddOrderBy("creationTime").
			Build()

		return s.upsertWithQuery(execCtx, query_, opt)
	}).WithMeta("operation", "upsertWith").
		ExecuteOrDefer(ctx)
}

// upsertWithQuery is the internal implementation for upsert
func (s *Service) upsertWithQuery(ctx context.Context, query string, opt CreateVersionOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.SoftwareVersion], bool) {
		// Search for existing version
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.SoftwareVersion]{}, false
		}

		// Check if any items were found
		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewSoftwareVersion(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		// Not found
		return op.Result[jsonmodels.SoftwareVersion]{}, false
	}

	// Define updater function
	updater := func(ctx context.Context, existing op.Result[jsonmodels.SoftwareVersion]) op.Result[jsonmodels.SoftwareVersion] {
		// Delete old binary if we're uploading a new one
		if opt.File.FilePath != "" || opt.URL != "" {
			s.deleteBinaryFromURL(ctx, existing.Data.URL())
		}

		// Upload new binary if needed
		url, err := s.uploadBinaryIfNeeded(ctx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](err, true)
		}

		// Build update body
		updateBody := map[string]any{
			"c8y_Software": map[string]any{
				"version": opt.Version,
			},
		}
		if url != "" {
			updateBody["c8y_Software"].(map[string]any)["url"] = url
		}

		// Update the version
		updateResult := s.Update(ctx, existing.Data.ID(), updateBody)
		if updateResult.Err != nil {
			return updateResult
		}
		return updateResult
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.SoftwareVersion] {
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

func (s *Service) createB(softwareID string, body any) *core.TryRequest {
	// Build request directly since childAdditions.createB is now private
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("id", softwareID).
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
