package firmwareitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/binaries"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamID = "id"
const ResultProperty = "managedObjects"
const FragmentFirmware = "c8y_Firmware"

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
	}
	service.Resolver = NewResolver(service)
	return service
}

// Service api to interact with firmware items
type Service struct {
	core.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	Resolver       *Resolver
}

type UploadFileOptions = core.UploadFileOptions

type CreateOptions struct {
	Name        string
	Description string
	URL         string
	DeviceType  string
	File        UploadFileOptions

	// AdditionalProperties allows adding custom fields to the managed object
	// These are merged into the body after standard fields are set
	// Standard fields (type, name, c8y_Filter, c8y_Global) cannot be overridden
	AdditionalProperties map[string]any
}

// Create a firmware item
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Firmware] {
	return op.Result[jsonmodels.Firmware]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Firmware] {
		// Build body - start with custom properties if provided
		body := make(map[string]any)

		// Merge custom properties first
		for k, v := range opt.AdditionalProperties {
			body[k] = v
		}

		// Set standard fields (these override any Properties values)
		body["type"] = FragmentFirmware
		body["name"] = opt.Name
		body["c8y_Global"] = map[string]any{}

		if opt.Description != "" {
			body["description"] = opt.Description
		}

		if opt.URL != "" {
			body["url"] = opt.URL
		}

		if opt.DeviceType != "" {
			body["c8y_Filter"] = map[string]any{
				"type": opt.DeviceType,
			}
		}

		// Use CreateWithBinary to handle file upload and child addition linking
		result := s.managedObjects.CreateWithBinary(execCtx, managedobjects.CreateWithBinaryOptions{
			Body:             body,
			File:             opt.File,
			SetURLField:      opt.URL == "" && opt.File.IsSet(), // Only set URL if not already provided
			URLFieldPath:     "url",
			AddChildAddition: opt.File.IsSet(),
		})

		if result.IsError() {
			return op.Failed[jsonmodels.Firmware](result.Err, result.IsRetryable())
		}

		// Convert ManagedObject result to Firmware result
		firmwareResult := op.Result[jsonmodels.Firmware]{
			Data:       jsonmodels.NewFirmware(result.Data.Bytes()),
			Status:     result.Status,
			HTTPStatus: result.HTTPStatus,
			Err:        result.Err,
			Meta:       result.Meta,
		}

		return firmwareResult
	}).WithMeta("operation", "create").
		ExecuteOrDefer(ctx)
}

// uploadBinaryIfNeeded uploads a binary file if needed, or returns the provided URL
func (s *Service) uploadBinaryIfNeeded(ctx context.Context, binaryUrl string, opt UploadFileOptions) (string, error) {
	// If URL is already provided, use it
	if binaryUrl != "" {
		return binaryUrl, nil
	}

	// If no file provided, return empty
	if opt.FilePath == "" {
		return "", nil
	}

	// Upload the file
	binaryResult := s.binaries.Create(ctx, opt)
	if binaryResult.IsError() {
		return "", fmt.Errorf("failed to upload binary: %w", binaryResult.Err)
	}

	return binaryResult.Data.Self(), nil
}

// extractIDFromURL extracts the ID from a Cumulocity URL
// Example: https://tenant.cumulocity.com/inventory/binaries/12345 -> "12345"
func extractIDFromURL(url string) string {
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// ResolveID resolves a firmware identifier to an ID using the resolver
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	return s.Resolver.ResolveID(ctx, identifier, meta)
}

// ListOptions filter firmware items
type ListOptions struct {
	Name       string `url:"-"`
	DeviceType string `url:"-"`
	pagination.PaginationOptions
}

// List firmware items
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Firmware] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewFirmware)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	query := model.NewInventoryQuery().
		AddOrderBy("name").
		AddOrderBy("creationTime").
		AddFilterEqStr("type", FragmentFirmware).
		AddFilterEqStr("name", opt.Name)

	if opt.DeviceType != "" {
		query.AddFilterEqStr("c8y_Filter.type", opt.DeviceType)
	}

	listOpts := managedobjects.ListOptions{
		Query: query.Build(),
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

// FirmwareIterator provides iteration over firmware items
type FirmwareIterator = pagination.Iterator[jsonmodels.Firmware]

// ListAll returns an iterator for all firmware items
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *FirmwareIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Firmware] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewFirmware,
	)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// DeleteOptions options to delete a firmware
type DeleteOptions struct {
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Resolver handles firmware resolution from various identifier formats
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

// ByName constructs a name-based reference
func (Ref) ByName(name string) string {
	return "name:" + name
}

// ByQuery constructs a custom query reference
func (Ref) ByQuery(query string) string {
	return "query:" + query
}

// NewResolver creates a new firmware resolver
func NewResolver(service *Service) *Resolver {
	return &Resolver{service: service}
}

// ByID returns the ID directly
func (r *Resolver) ByID(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("id cannot be empty")
	}
	return id, nil
}

// ByName resolves a firmware by name
func (r *Resolver) ByName(ctx context.Context, name string) (string, error) {
	return r.resolveByName(ctx, name)
}

// ByQuery resolves a firmware using a custom query
func (r *Resolver) ByQuery(ctx context.Context, query string) (string, error) {
	return r.resolveByQuery(ctx, query)
}

// ResolveID resolves a firmware identifier string to an ID
func (r *Resolver) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	if meta == nil {
		meta = make(map[string]any)
	}

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
	case "name":
		name := parts[1]
		meta["resolverType"] = "name"
		meta["name"] = name
		return r.resolveByName(ctx, name)

	case "query":
		query := strings.Join(parts[1:], ":")
		meta["resolverType"] = "query"
		meta["query"] = query
		return r.resolveByQuery(ctx, query)

	default:
		return "", fmt.Errorf("unsupported resolver type: %s", resolverType)
	}
}

// resolveByName resolves by name
func (r *Resolver) resolveByName(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		Name: name,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup firmware: %w", listResult.Err)
	}

	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewFirmware(item.Bytes())
		return found.ID(), nil
	}

	return "", fmt.Errorf("firmware not found: name=%s", name)
}

// resolveByQuery resolves by custom query
func (r *Resolver) resolveByQuery(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	fullQuery := model.NewInventoryQuery().
		AddFilterEqStr("type", FragmentFirmware).
		AddFilterPart(query).
		Build()

	moResult := r.service.managedObjects.List(ctx, managedobjects.ListOptions{
		Query: fullQuery,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if moResult.Err != nil {
		return "", fmt.Errorf("failed to lookup firmware: %w", moResult.Err)
	}

	for item := range moResult.Data.Iter() {
		found := jsonmodels.NewFirmware(item.Bytes())
		return found.ID(), nil
	}

	return "", fmt.Errorf("firmware not found: query=%s", query)
}

// Get retrieves a firmware
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.Firmware] {
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Firmware](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewFirmware, meta)
}

// Update a firmware
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Firmware] {
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Firmware](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewFirmware, meta)
}

// Delete a firmware
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
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

// GetOrCreate searches by name, creating if not found
func (s *Service) GetOrCreate(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Firmware] {
	return op.Result[jsonmodels.Firmware]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Firmware] {
		finder := func(ctx context.Context) (op.Result[jsonmodels.Firmware], bool) {
			listResult := s.List(ctx, ListOptions{
				Name: opt.Name,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.Firmware]{}, false
			}

			for item := range listResult.Data.Iter() {
				found := jsonmodels.NewFirmware(item.Bytes())
				result := op.OK(found)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["lookupMethod"] = "name"
				return result, true
			}

			return op.Result[jsonmodels.Firmware]{}, false
		}

		creator := func(ctx context.Context) op.Result[jsonmodels.Firmware] {
			return s.Create(ctx, opt)
		}

		return op.GetOrCreateR(execCtx, finder, creator)
	}).WithMeta("operation", "getOrCreate").
		ExecuteOrDefer(ctx)
}

// UpsertByName searches by name, updating if found or creating if not found
func (s *Service) UpsertByName(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Firmware] {
	return op.Result[jsonmodels.Firmware]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Firmware] {
		finder := func(ctx context.Context) (op.Result[jsonmodels.Firmware], bool) {
			listResult := s.List(ctx, ListOptions{
				Name: opt.Name,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.Firmware]{}, false
			}

			for item := range listResult.Data.Iter() {
				found := jsonmodels.NewFirmware(item.Bytes())
				result := op.OK(found)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["lookupMethod"] = "name"
				return result, true
			}

			return op.Result[jsonmodels.Firmware]{}, false
		}

		updater := func(ctx context.Context, existing op.Result[jsonmodels.Firmware]) op.Result[jsonmodels.Firmware] {
			updateBody := map[string]any{}

			if opt.Description != "" {
				updateBody["description"] = opt.Description
			}

			// Handle binary upload and child addition linking
			if opt.URL != "" || opt.File.FilePath != "" {
				url, err := s.uploadBinaryIfNeeded(ctx, opt.URL, opt.File)
				if err != nil {
					return op.Failed[jsonmodels.Firmware](err, true)
				}
				if url != "" {
					updateBody["url"] = url

					// If a new binary was uploaded (not just URL provided), link it as child addition
					if opt.File.FilePath != "" {
						// Extract binary ID from URL (last segment)
						// URL format: https://.../inventory/binaries/{id}
						binaryID := extractIDFromURL(url)
						if binaryID != "" {
							additionResult := s.managedObjects.ChildAdditions.Create(ctx, existing.Data.ID(), binaryID)
							if additionResult.IsError() {
								// Don't fail the update if child addition fails, just note in meta
								updateBody["_childAdditionError"] = additionResult.Err.Error()
							}
						}
					}
				}
			}

			if opt.DeviceType != "" {
				updateBody["c8y_Filter"] = map[string]any{
					"type": opt.DeviceType,
				}
			}

			if len(updateBody) == 0 {
				return existing
			}

			return s.Update(ctx, existing.Data.ID(), updateBody)
		}

		creator := func(ctx context.Context) op.Result[jsonmodels.Firmware] {
			return s.Create(ctx, opt)
		}

		return op.UpsertR(execCtx, finder, updater, creator)
	}).WithMeta("operation", "upsertByName").
		ExecuteOrDefer(ctx)
}

// Builder methods

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetContentType(types.MimeTypeManagedObject).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	getOpts := managedobjects.GetOptions{
		WithParents: opt.WithParents,
	}
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(getOpts)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", ID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	deleteOpts := managedobjects.DeleteOptions{
		ForceCascade: opt.ForceCascade,
	}
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(deleteOpts)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req, "")
}
