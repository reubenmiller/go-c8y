package firmwarepatches

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

const ApiManagedObjects = "/inventory/managedObjects"
const ApiManagedObject = "/inventory/managedObjects/{id}"
const ParamId = "id"
const ResultProperty = "managedObjects"

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

// Service provides firmware patch operations
type Service struct {
	core.Service
	firmware       *firmwareitems.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	Resolver       *Resolver
}

type CreateOptions struct {
	FirmwareID        string
	Version           string
	DependencyVersion string
	URL               string
	File              core.UploadFileOptions
}

// Create creates a firmware patch
func (s *Service) Create(ctx context.Context, firmwareID string, opt CreateOptions) op.Result[jsonmodels.FirmwarePatch] {
	return op.Result[jsonmodels.FirmwarePatch]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwarePatch] {
		resolvedID, err := s.firmware.Resolver.ResolveID(execCtx, firmwareID, nil)
		if err != nil {
			return op.Failed[jsonmodels.FirmwarePatch](
				fmt.Errorf("failed to resolve firmware ID: %w", err),
				false,
			)
		}

		url, err := s.uploadBinaryIfNeeded(execCtx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.FirmwarePatch](err, true)
		}

		patchBody := map[string]any{
			"type": "c8y_FirmwareBinary",
			"c8y_Firmware": map[string]any{
				"version": opt.Version,
			},
			"c8y_Patch": map[string]any{
				"dependency": opt.DependencyVersion,
			},
			"c8y_Global": map[string]any{},
		}
		if url != "" {
			patchBody["c8y_Firmware"].(map[string]any)["url"] = url
		}

		return core.Execute(execCtx, s.createB(resolvedID, patchBody), jsonmodels.NewFirmwarePatch)
	}).WithMeta("operation", "create").
		ExecuteOrDefer(ctx)
}

type ListOptions struct {
	FirmwareID        string `url:"-"`
	Version           string `url:"-"`
	DependencyVersion string `url:"-"`
	URL               string `url:"-"`
	Query             string `url:"-"`
	pagination.PaginationOptions
}

// List lists firmware patches
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.FirmwarePatch] {
	firmwareResult := s.firmware.Get(ctx, opt.FirmwareID, firmwareitems.GetOptions{})
	if firmwareResult.Err != nil {
		return op.Failed[jsonmodels.FirmwarePatch](
			fmt.Errorf("failed to resolve firmware: %w", firmwareResult.Err),
			true,
		)
	}
	opt.FirmwareID = firmwareResult.Data.ID()

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewFirmwarePatch)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	query := model.NewInventoryQuery().
		AddFilterEqStr("type", "c8y_FirmwareBinary").
		AddFilterEqStr("c8y_Firmware.version", opt.Version).
		AddFilterEqStr("c8y_Patch.dependency", opt.DependencyVersion).
		AddFilterEqStr("c8y_Firmware.url", opt.URL).
		AddFilterPart(opt.Query).
		ByGroupID(opt.FirmwareID).
		AddOrderBy("c8y_Firmware.version").
		AddOrderBy("creationTime").
		HasFragment("c8y_Patch")

	listOpts := managedobjects.ListOptions{
		Query: query.Build(),
		PaginationOptions: pagination.PaginationOptions{
			CurrentPage: opt.CurrentPage,
			PageSize:    opt.PageSize,
		},
	}
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(listOpts)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.managedObjects.Client, req, managedobjects.ResultProperty)
}

// ListAll returns an iterator for all firmware patches
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *pagination.Iterator[jsonmodels.FirmwarePatch] {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.FirmwarePatch] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewFirmwarePatch,
	)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// Get retrieves a firmware patch
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.FirmwarePatch] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.FirmwarePatch](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewFirmwarePatch, meta)
}

// Update updates a firmware patch
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.FirmwarePatch] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.FirmwarePatch](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewFirmwarePatch, meta)
}

type DeleteOptions struct {
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Delete deletes a firmware patch
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.FirmwarePatch] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.FirmwarePatch](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.deleteB(id, opt), jsonmodels.NewFirmwarePatch, meta).IgnoreNotFound()
}

// Resolver handles firmware patch resolution
type Resolver struct {
	service *Service
}

// NewResolver creates a new firmware patch resolver
func NewResolver(service *Service) *Resolver {
	return &Resolver{service: service}
}

// Ref provides helper methods to construct resolver identifiers
type Ref struct{}

// NewRef creates a new Ref
func NewRef() *Ref {
	return &Ref{}
}

// ByID constructs a direct ID reference
func (Ref) ByID(id string) string {
	return id
}

// ByVersion constructs a version-based reference using firmware ID
func (Ref) ByVersion(version, firmwareID string) string {
	return "version:" + version + ":firmware:" + firmwareID
}

// ByVersionAndName constructs a version-based reference using firmware name
func (Ref) ByVersionAndName(version, firmwareName string) string {
	return "version:" + version + ":name:" + firmwareName
}

// ByVersionAndDependency constructs a version with dependency reference
func (Ref) ByVersionAndDependency(version, dependency, firmwareID string) string {
	return "version:" + version + ":dependency:" + dependency + ":firmware:" + firmwareID
}

// ResolveID resolves a firmware patch identifier to an ID
// Supported formats:
//   - "12345" - direct ID
//   - "version:1.0.1:firmware:12345" - by version and firmware ID
//   - "version:1.0.1:name:MyFirmware" - by version and firmware name
//   - "version:1.0.1:dependency:1.0.0:firmware:12345" - by version, dependency and firmware ID
func (r *Resolver) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	if meta == nil {
		meta = make(map[string]any)
	}

	if identifier == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}

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
		if len(parts) < 4 {
			return "", fmt.Errorf("version resolver requires format 'version:VERSION:firmware:ID' or 'version:VERSION:name:NAME': %s", identifier)
		}
		version := parts[1]
		lookupType := parts[2]

		meta["resolverType"] = "version"
		meta["version"] = version

		switch lookupType {
		case "firmware":
			firmwareID := parts[3]
			meta["firmwareID"] = firmwareID
			return r.resolveByVersionAndFirmwareID(ctx, version, "", firmwareID)

		case "name":
			firmwareName := parts[3]
			meta["firmwareName"] = firmwareName
			return r.resolveByVersionAndFirmwareName(ctx, version, "", firmwareName)

		case "dependency":
			if len(parts) < 6 {
				return "", fmt.Errorf("dependency resolver requires format 'version:VERSION:dependency:DEP:firmware:ID': %s", identifier)
			}
			dependency := parts[3]
			lookupType2 := parts[4]
			lookupValue2 := parts[5]

			meta["dependency"] = dependency

			switch lookupType2 {
			case "firmware":
				meta["firmwareID"] = lookupValue2
				return r.resolveByVersionAndFirmwareID(ctx, version, dependency, lookupValue2)

			case "name":
				meta["firmwareName"] = lookupValue2
				return r.resolveByVersionAndFirmwareName(ctx, version, dependency, lookupValue2)

			default:
				return "", fmt.Errorf("unsupported dependency lookup type: %s", lookupType2)
			}

		default:
			return "", fmt.Errorf("unsupported version lookup type: %s", lookupType)
		}

	default:
		return "", fmt.Errorf("unsupported resolver type: %s", resolverType)
	}
}

func (r *Resolver) resolveByVersionAndFirmwareID(ctx context.Context, version, dependency, firmwareID string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if firmwareID == "" {
		return "", fmt.Errorf("firmwareID cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		FirmwareID:        firmwareID,
		Version:           version,
		DependencyVersion: dependency,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup patch: %w", listResult.Err)
	}

	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewFirmwarePatch(item.Bytes())
		return found.ID(), nil
	}

	if dependency != "" {
		return "", fmt.Errorf("patch not found: firmware=%s, version=%s, dependency=%s", firmwareID, version, dependency)
	}
	return "", fmt.Errorf("patch not found: firmware=%s, version=%s", firmwareID, version)
}

func (r *Resolver) resolveByVersionAndFirmwareName(ctx context.Context, version, dependency, firmwareName string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}
	if firmwareName == "" {
		return "", fmt.Errorf("firmware name cannot be empty")
	}

	identifier := "name:" + firmwareName
	firmwareResult := r.service.firmware.Get(ctx, identifier, firmwareitems.GetOptions{})
	if firmwareResult.Err != nil {
		return "", fmt.Errorf("failed to resolve firmware name: %w", firmwareResult.Err)
	}

	firmwareID := firmwareResult.Data.ID()
	return r.resolveByVersionAndFirmwareID(ctx, version, dependency, firmwareID)
}

type CreatePatchOptions struct {
	FirmwareName      string
	FirmwareID        string
	Version           string
	DependencyVersion string
	URL               string
	File              core.UploadFileOptions
}

// CreatePatch creates a firmware patch with automatic firmware resolution
func (s *Service) CreatePatch(ctx context.Context, opt CreatePatchOptions) op.Result[jsonmodels.FirmwarePatch] {
	return op.Result[jsonmodels.FirmwarePatch]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwarePatch] {
		var firmwareID string
		if opt.FirmwareID != "" {
			firmwareID = opt.FirmwareID
		} else if opt.FirmwareName != "" {
			firmwareResult := s.firmware.Get(execCtx, firmwareitems.NewRef().ByName(opt.FirmwareName), firmwareitems.GetOptions{})
			if firmwareResult.Err != nil {
				return op.Failed[jsonmodels.FirmwarePatch](
					fmt.Errorf("failed to resolve firmware: %w", firmwareResult.Err),
					true,
				)
			}
			firmwareID = firmwareResult.Data.ID()
		} else {
			return op.Failed[jsonmodels.FirmwarePatch](
				fmt.Errorf("must specify FirmwareID or FirmwareName"),
				false,
			)
		}

		createOpt := CreateOptions{
			Version:           opt.Version,
			DependencyVersion: opt.DependencyVersion,
			URL:               opt.URL,
			File:              opt.File,
		}
		return s.Create(execCtx, firmwareID, createOpt)
	}).WithMeta("operation", "createPatch").
		ExecuteOrDefer(ctx)
}

// GetOrCreatePatch gets or creates a firmware patch
func (s *Service) GetOrCreatePatch(ctx context.Context, opt CreatePatchOptions) op.Result[jsonmodels.FirmwarePatch] {
	return op.Result[jsonmodels.FirmwarePatch]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwarePatch] {
		finder := func(ctx context.Context) (op.Result[jsonmodels.FirmwarePatch], bool) {
			firmwareID := opt.FirmwareID
			if firmwareID == "" && opt.FirmwareName != "" {
				firmwareResult := s.firmware.Get(ctx, firmwareitems.NewRef().ByName(opt.FirmwareName), firmwareitems.GetOptions{})
				if firmwareResult.Err != nil {
					return op.Result[jsonmodels.FirmwarePatch]{}, false
				}
				firmwareID = firmwareResult.Data.ID()
			}

			if firmwareID == "" {
				return op.Result[jsonmodels.FirmwarePatch]{}, false
			}

			listResult := s.List(ctx, ListOptions{
				FirmwareID:        firmwareID,
				Version:           opt.Version,
				DependencyVersion: opt.DependencyVersion,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.FirmwarePatch]{}, false
			}

			for item := range listResult.Data.Iter() {
				patch := jsonmodels.NewFirmwarePatch(item.Bytes())
				result := op.OK(patch)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["found"] = true
				result.Meta["lookupMethod"] = "patch"
				return result, true
			}

			return op.Result[jsonmodels.FirmwarePatch]{}, false
		}

		creator := func(ctx context.Context) op.Result[jsonmodels.FirmwarePatch] {
			createResult := s.CreatePatch(ctx, opt)
			if createResult.Err != nil {
				return createResult
			}
			createResult.Meta["found"] = false
			return createResult
		}

		return op.GetOrCreateR(execCtx, finder, creator)
	}).WithMeta("operation", "getOrCreatePatch").
		ExecuteOrDefer(ctx)
}

// UpsertByVersionAndDependency upserts a firmware patch
func (s *Service) UpsertByVersionAndDependency(ctx context.Context, opt CreatePatchOptions) op.Result[jsonmodels.FirmwarePatch] {
	return op.Result[jsonmodels.FirmwarePatch]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.FirmwarePatch] {
		firmwareID := opt.FirmwareID
		if firmwareID == "" && opt.FirmwareName != "" {
			firmwareResult := s.firmware.Get(execCtx, firmwareitems.NewRef().ByName(opt.FirmwareName), firmwareitems.GetOptions{})
			if firmwareResult.Err != nil {
				return op.Failed[jsonmodels.FirmwarePatch](
					fmt.Errorf("failed to resolve firmware: %w", firmwareResult.Err),
					true,
				)
			}
			firmwareID = firmwareResult.Data.ID()
		}

		if firmwareID == "" {
			return op.Failed[jsonmodels.FirmwarePatch](
				fmt.Errorf("must specify FirmwareID or FirmwareName"),
				false,
			)
		}

		query := model.NewInventoryQuery().
			AddFilterEqStr("type", "c8y_FirmwareBinary").
			AddFilterEqStr("c8y_Firmware.version", opt.Version).
			AddFilterEqStr("c8y_Patch.dependency", opt.DependencyVersion).
			ByGroupID(firmwareID).
			HasFragment("c8y_Patch").
			Build()

		return s.upsertWithQuery(execCtx, query, opt)
	}).WithMeta("operation", "upsertByVersionAndDependency").
		ExecuteOrDefer(ctx)
}

func (s *Service) upsertWithQuery(ctx context.Context, query string, opt CreatePatchOptions) op.Result[jsonmodels.FirmwarePatch] {
	finder := func(ctx context.Context) (op.Result[jsonmodels.FirmwarePatch], bool) {
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.FirmwarePatch]{}, false
		}

		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewFirmwarePatch(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		return op.Result[jsonmodels.FirmwarePatch]{}, false
	}

	updater := func(ctx context.Context, existing op.Result[jsonmodels.FirmwarePatch]) op.Result[jsonmodels.FirmwarePatch] {
		if opt.File.FilePath != "" || opt.URL != "" {
			s.deleteBinaryFromURL(ctx, existing.Data.URL())
		}

		url, err := s.uploadBinaryIfNeeded(ctx, opt.URL, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.FirmwarePatch](err, true)
		}

		updateBody := map[string]any{
			"c8y_Firmware": map[string]any{
				"version": opt.Version,
			},
			"c8y_Patch": map[string]any{
				"dependency": opt.DependencyVersion,
			},
		}
		if url != "" {
			updateBody["c8y_Firmware"].(map[string]any)["url"] = url
		}

		updateResult := s.Update(ctx, existing.Data.ID(), updateBody)
		if updateResult.Err != nil {
			return updateResult
		}
		return updateResult
	}

	creator := func(ctx context.Context) op.Result[jsonmodels.FirmwarePatch] {
		return s.CreatePatch(ctx, opt)
	}

	return op.UpsertR(ctx, finder, updater, creator)
}

func extractBinaryID(url string) string {
	if strings.HasPrefix(url, "/inventory/binaries/") {
		parts := strings.Split(url, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
	}

	if strings.Contains(url, "/inventory/binaries/") {
		parts := strings.Split(url, "/inventory/binaries/")
		if len(parts) == 2 {
			binaryID := strings.Split(parts[1], "?")[0]
			return binaryID
		}
	}

	return ""
}

func (s *Service) uploadBinaryIfNeeded(ctx context.Context, binaryUrl string, opt core.UploadFileOptions) (string, error) {
	if binaryUrl != "" {
		return binaryUrl, nil
	}

	binaryResult := s.binaries.Create(ctx, opt)
	if binaryResult.IsError() {
		return "", fmt.Errorf("failed to upload binary: %w", binaryResult.Err)
	}

	return binaryResult.Data.Self(), nil
}

func (s *Service) deleteBinaryFromURL(ctx context.Context, url string) {
	if url == "" {
		return
	}

	binaryID := extractBinaryID(url)
	if binaryID == "" {
		return
	}

	deleteResult := s.binaries.Delete(ctx, binaryID)
	if deleteResult.Err != nil {
		slog.Info("failed to delete old binary", "binaryID", binaryID, "err", deleteResult.Err)
	}
}

func (s *Service) createB(firmwareID string, body any) *core.TryRequest {
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("id", firmwareID).
		SetBody(body).
		SetContentType(types.MimeTypeManagedObject).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/inventory/managedObjects/{id}/childAdditions")
	return core.NewTryRequest(s.managedObjects.Client, req, "")
}

func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	getOpts := managedobjects.GetOptions{
		WithParents: opt.WithParents,
	}
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(getOpts)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.managedObjects.Client, req, "")
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.managedObjects.Client, req, "")
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	deleteOpts := managedobjects.DeleteOptions{
		ForceCascade: opt.ForceCascade,
	}
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(deleteOpts)).
		SetURL(managedobjects.ApiManagedObject)
	return core.NewTryRequest(s.managedObjects.Client, req, "")
}
