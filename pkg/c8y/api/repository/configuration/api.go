package configuration

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

const ParamId = "id"
const ResultProperty = "managedObjects"
const FragmentConfiguration = "c8y_ConfigurationDump"

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
	}
	service.Resolver = NewResolver(service)
	return service
}

// Service api to interact with configuration items
type Service struct {
	core.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	Resolver       *Resolver
}

type UploadFileOptions = core.UploadFileOptions

type CreateOptions struct {
	Name              string
	ConfigurationType string
	Description       string
	URL               string
	DeviceType        string
	File              UploadFileOptions

	// AdditionalProperties allows adding custom fields to the managed object
	// These are merged into the body after standard fields are set
	// Standard fields (type, name, configurationType, c8y_Global) cannot be overridden
	AdditionalProperties map[string]any
}

// Create a configuration item
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Configuration] {
	return op.Result[jsonmodels.Configuration]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Configuration] {
		// Build body - start with custom properties if provided
		body := make(map[string]any)

		// Merge custom properties first
		for k, v := range opt.AdditionalProperties {
			body[k] = v
		}

		// Set standard fields (these override any Properties values)
		body["type"] = FragmentConfiguration
		body["name"] = opt.Name
		body["configurationType"] = opt.ConfigurationType
		body["c8y_Global"] = map[string]any{}

		if opt.Description != "" {
			body["description"] = opt.Description
		}

		if opt.URL != "" {
			body["url"] = opt.URL
		}

		if opt.DeviceType != "" {
			body["deviceType"] = opt.DeviceType
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
			return op.Failed[jsonmodels.Configuration](result.Err, result.IsRetryable())
		}

		// Convert ManagedObject result to Configuration result
		configResult := op.Result[jsonmodels.Configuration]{
			Data:       jsonmodels.NewConfiguration(result.Data.Bytes()),
			Status:     result.Status,
			HTTPStatus: result.HTTPStatus,
			Err:        result.Err,
			Meta:       result.Meta,
		}

		return configResult
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

// ResolveID resolves a configuration identifier to an ID using the resolver
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	return s.Resolver.ResolveID(ctx, identifier, meta)
}

// ListOptions filter configurations
type ListOptions struct {
	Name              string `url:"-"`
	ConfigurationType string `url:"-"`
	DeviceType        string `url:"-"`
	pagination.PaginationOptions
}

// List configurations
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Configuration] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewConfiguration)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddOrderBy("name").
			AddOrderBy("creationTime").
			AddFilterEqStr("type", FragmentConfiguration).
			AddFilterEqStr("name", opt.Name).
			AddFilterEqStr("configurationType", opt.ConfigurationType).
			AddFilterEqStr("deviceType", opt.DeviceType).
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

// ConfigurationIterator provides iteration over configuration items
type ConfigurationIterator = pagination.Iterator[jsonmodels.Configuration]

// ListAll returns an iterator for all configuration items
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ConfigurationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Configuration] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewConfiguration,
	)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// DeleteOptions options to delete a configuration
type DeleteOptions struct {
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Resolver handles configuration resolution from various identifier formats
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

// ByName constructs a name-based reference with optional configuration type
func (Ref) ByName(name string, configurationType ...string) string {
	if len(configurationType) > 0 && configurationType[0] != "" {
		return "name:" + name + ":" + configurationType[0]
	}
	return "name:" + name
}

// ByQuery constructs a custom query reference
func (Ref) ByQuery(query string) string {
	return "query:" + query
}

// NewResolver creates a new configuration resolver
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

// ByName resolves a configuration by name and optional configuration type
func (r *Resolver) ByName(ctx context.Context, name, configurationType string) (string, error) {
	return r.resolveByNameAndType(ctx, name, configurationType)
}

// ByQuery resolves a configuration using a custom query
func (r *Resolver) ByQuery(ctx context.Context, query string) (string, error) {
	return r.resolveByQuery(ctx, query)
}

// ResolveID resolves a configuration identifier string to an ID
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
		configurationType := ""
		if len(parts) >= 3 {
			configurationType = parts[2]
		}
		meta["resolverType"] = "name"
		meta["name"] = name
		if configurationType != "" {
			meta["configurationType"] = configurationType
		}
		return r.resolveByNameAndType(ctx, name, configurationType)

	case "query":
		query := strings.Join(parts[1:], ":")
		meta["resolverType"] = "query"
		meta["query"] = query
		return r.resolveByQuery(ctx, query)

	default:
		return "", fmt.Errorf("unsupported resolver type: %s", resolverType)
	}
}

// resolveByNameAndType resolves by name and optional configuration type
func (r *Resolver) resolveByNameAndType(ctx context.Context, name, configurationType string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		Name:              name,
		ConfigurationType: configurationType,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup configuration: %w", listResult.Err)
	}

	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewConfiguration(item.Bytes())
		return found.ID(), nil
	}

	if configurationType != "" {
		return "", fmt.Errorf("configuration not found: name=%s, configurationType=%s", name, configurationType)
	}
	return "", fmt.Errorf("configuration not found: name=%s", name)
}

// resolveByQuery resolves by custom query
func (r *Resolver) resolveByQuery(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	fullQuery := model.NewInventoryQuery().
		AddFilterEqStr("type", FragmentConfiguration).
		AddFilterPart(query).
		Build()

	moResult := r.service.managedObjects.List(ctx, managedobjects.ListOptions{
		Query: fullQuery,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if moResult.Err != nil {
		return "", fmt.Errorf("failed to lookup configuration: %w", moResult.Err)
	}

	for item := range moResult.Data.Iter() {
		found := jsonmodels.NewConfiguration(item.Bytes())
		return found.ID(), nil
	}

	return "", fmt.Errorf("configuration not found: query=%s", query)
}

// Get retrieves a configuration
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.Configuration] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Configuration](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewConfiguration, meta)
}

// Update a configuration
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Configuration] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Configuration](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewConfiguration, meta)
}

// Delete a configuration
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	meta["id"] = id

	return core.ExecuteNoContent(ctx, s.deleteB(id, opt), meta).IgnoreNotFound()
}

// GetOrCreate searches by name and optional configuration type, creating if not found
func (s *Service) GetOrCreate(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Configuration] {
	return op.Result[jsonmodels.Configuration]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Configuration] {
		finder := func(ctx context.Context) (op.Result[jsonmodels.Configuration], bool) {
			listResult := s.List(ctx, ListOptions{
				Name:              opt.Name,
				ConfigurationType: opt.ConfigurationType,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.Configuration]{}, false
			}

			for item := range listResult.Data.Iter() {
				found := jsonmodels.NewConfiguration(item.Bytes())
				result := op.OK(found)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["lookupMethod"] = "name"
				return result, true
			}

			return op.Result[jsonmodels.Configuration]{}, false
		}

		creator := func(ctx context.Context) op.Result[jsonmodels.Configuration] {
			return s.Create(ctx, opt)
		}

		return op.GetOrCreateR(execCtx, finder, creator)
	}).WithMeta("operation", "getOrCreate").
		ExecuteOrDefer(ctx)
}

// UpsertByName searches by name and optional configuration type, updating if found or creating if not found
func (s *Service) UpsertByName(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Configuration] {
	return op.Result[jsonmodels.Configuration]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Configuration] {
		finder := func(ctx context.Context) (op.Result[jsonmodels.Configuration], bool) {
			listResult := s.List(ctx, ListOptions{
				Name:              opt.Name,
				ConfigurationType: opt.ConfigurationType,
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})

			if listResult.Err != nil {
				return op.Result[jsonmodels.Configuration]{}, false
			}

			for item := range listResult.Data.Iter() {
				found := jsonmodels.NewConfiguration(item.Bytes())
				result := op.OK(found)
				result.HTTPStatus = listResult.HTTPStatus
				result.Meta["lookupMethod"] = "name"
				return result, true
			}

			return op.Result[jsonmodels.Configuration]{}, false
		}

		updater := func(ctx context.Context, existing op.Result[jsonmodels.Configuration]) op.Result[jsonmodels.Configuration] {
			updateBody := map[string]any{}

			if opt.Description != "" {
				updateBody["description"] = opt.Description
			}

			// Handle binary upload and child addition linking
			if opt.URL != "" || opt.File.FilePath != "" {
				url, err := s.uploadBinaryIfNeeded(ctx, opt.URL, opt.File)
				if err != nil {
					return op.Failed[jsonmodels.Configuration](err, true)
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
				updateBody["deviceType"] = opt.DeviceType
			}

			if len(updateBody) == 0 {
				return existing
			}

			return s.Update(ctx, existing.Data.ID(), updateBody)
		}

		creator := func(ctx context.Context) op.Result[jsonmodels.Configuration] {
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
