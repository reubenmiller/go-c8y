package softwareitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
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
		managedObjects: managedobjects.NewService(s),
	}
	service.Resolver = NewResolver(service)
	return service
}

// Service api to interact with software items
type Service struct {
	core.Service
	managedObjects *managedobjects.Service
	Resolver       *Resolver
}

// Create a software item
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Software] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewSoftware)
}

// ResolveID resolves a software identifier to an ID using the resolver
// This is a convenience method that wraps the Resolver.ResolveID
// Supported formats:
//   - "12345" - direct ID
//   - "name:MySoftware" - lookup by name
//   - "name:MySoftware:application" - lookup by name and softwareType
//   - "query:name eq 'MySoftware'" - custom query
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	return s.Resolver.ResolveID(ctx, identifier, meta)
}

// ListOptions filter software
type ListOptions struct {
	Name string `url:"-"`

	SoftwareType string `url:"-"`

	DeviceType string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List software
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Software] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSoftware)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	// Build request directly since managedObjects.listB is now private
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddOrderBy("name").
			AddOrderBy("creationTime").
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", opt.Name).
			AddFilterEqStr("softwareType", opt.SoftwareType).
			AddFilterEqStr("c8y_Filter.type", opt.DeviceType).
			Build(),
		PaginationOptions: pagination.PaginationOptions{
			CurrentPage: opt.CurrentPage,
			PageSize:    opt.PageSize,
		},
	}
	// Build the request directly
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(listOpts)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.managedObjects.Client, req, managedobjects.ResultProperty)
}

// SoftwareIterator provides iteration over software items
type SoftwareIterator = pagination.Iterator[jsonmodels.Software]

// ListAll returns an iterator for all software items
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SoftwareIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Software] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewSoftware,
	)
}

type GetOptions struct {
	// Query options
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// DeleteOptions options to delete a software item
type DeleteOptions struct {
	// Delete options
	SkipCascade bool `url:"-"`
}

// Resolver handles software item resolution from various identifier formats
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

// ByName constructs a name-based reference with optional software type
// Examples:
//   - Ref{}.ByName("MySoftware") -> "name:MySoftware"
//   - Ref{}.ByName("MySoftware", "application") -> "name:MySoftware:application"
func (Ref) ByName(name string, softwareType ...string) string {
	if len(softwareType) > 0 && softwareType[0] != "" {
		return "name:" + name + ":" + softwareType[0]
	}
	return "name:" + name
}

// ByQuery constructs a query-based reference
func (Ref) ByQuery(query string) string {
	return "query:" + query
}

// NewResolver creates a new software resolver
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

// ByName resolves software by name (optionally with softwareType)
func (r *Resolver) ByName(ctx context.Context, name, softwareType string) (string, error) {
	return r.resolveByNameAndType(ctx, name, softwareType)
}

// ByQuery resolves software using a custom query
func (r *Resolver) ByQuery(ctx context.Context, query string) (string, error) {
	meta := make(map[string]any)
	return r.resolveByQuery(ctx, query, meta)
}

// ResolveID resolves a software identifier string to an ID
// Supported formats:
//   - "12345" - direct ID
//   - "name:MySoftware" - lookup by name
//   - "name:MySoftware:application" - lookup by name and softwareType
//   - "query:name eq 'MySoftware'" - custom query
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

	parts := strings.SplitN(identifier, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid identifier format: %s", identifier)
	}

	resolverType := parts[0]
	resolverValue := parts[1]

	switch resolverType {
	case "name":
		// Format: "name:MySoftware" or "name:MySoftware:application"
		return r.resolveByName(ctx, resolverValue, meta)

	case "query":
		// Format: "query:name eq 'MySoftware' and softwareType eq 'application'"
		return r.resolveByQuery(ctx, resolverValue, meta)

	default:
		return "", fmt.Errorf("unsupported resolver type: %s", resolverType)
	}
}

// resolveByName resolves by name, optionally with softwareType
// Format: "MySoftware" or "MySoftware:application"
func (r *Resolver) resolveByName(ctx context.Context, nameSpec string, meta map[string]any) (string, error) {
	var name, softwareType string

	// Check if softwareType is included
	if strings.Contains(nameSpec, ":") {
		parts := strings.SplitN(nameSpec, ":", 2)
		name = parts[0]
		softwareType = parts[1]
	} else {
		name = nameSpec
	}

	meta["name"] = name
	if softwareType != "" {
		meta["resolverType"] = "nameAndType"
		meta["softwareType"] = softwareType
	} else {
		meta["resolverType"] = "name"
	}

	return r.resolveByNameAndType(ctx, name, softwareType)
}

// resolveByNameAndType resolves software by name and optional type
func (r *Resolver) resolveByNameAndType(ctx context.Context, name, softwareType string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	listResult := r.service.List(ctx, ListOptions{
		Name:         name,
		SoftwareType: softwareType,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup software by name: %w", listResult.Err)
	}

	// Check if any items were found
	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewSoftware(item.Bytes())
		return found.ID(), nil
	}

	if softwareType != "" {
		return "", fmt.Errorf("software not found with name=%s, softwareType=%s", name, softwareType)
	}
	return "", fmt.Errorf("software not found with name=%s", name)
}

// resolveByQuery resolves using a custom query
func (r *Resolver) resolveByQuery(ctx context.Context, query string, meta map[string]any) (string, error) {
	meta["resolverType"] = "query"
	meta["query"] = query

	// Build full query with software type filter
	fullQuery := model.NewInventoryQuery().
		AddFilterEqStr("type", FragmentSoftware).
		AddFilterPart(query).
		Build()

	listResult := r.service.managedObjects.List(ctx, managedobjects.ListOptions{
		Query: fullQuery,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 1,
		},
	})

	if listResult.Err != nil {
		return "", fmt.Errorf("failed to lookup software by query: %w", listResult.Err)
	}

	// Check if any items were found
	for item := range listResult.Data.Iter() {
		found := jsonmodels.NewSoftware(item.Bytes())
		return found.ID(), nil
	}

	return "", fmt.Errorf("software not found with query: %s", query)
}

// Get retrieves a software item
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "name:MySoftware" - lookup by name
//   - "name:MySoftware:application" - lookup by name and softwareType
//   - "query:name eq 'MySoftware'" - custom query
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.Software] {
	// Resolve ID (supports "name:MySoftware", "name:MySoftware:application", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Use background context for resolution so it doesn't inherit the deferred flag
		// This allows lookups (like List) to actually execute
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Software](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.getB(id, opt), jsonmodels.NewSoftware, meta)
}

// Update a software item
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "name:MySoftware" - lookup by name
//   - "name:MySoftware:application" - lookup by name and softwareType
//   - "query:name eq 'MySoftware'" - custom query
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Software] {
	// Resolve ID (supports "name:MySoftware", "name:MySoftware:application", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Software](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewSoftware, meta)
}

// Delete a software item
// ID supports both direct IDs and string-based resolver patterns:
//   - "12345" - direct ID
//   - "name:MySoftware" - lookup by name
//   - "name:MySoftware:application" - lookup by name and softwareType
//   - "query:name eq 'MySoftware'" - custom query
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.Software] {
	// Resolve ID (supports "name:MySoftware", "name:MySoftware:application", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	meta["identifier"] = ID
	id, err := s.Resolver.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.Software](err, false)
	}
	meta["id"] = id

	return core.Execute(ctx, s.deleteB(id, opt), jsonmodels.NewSoftware, meta)
}

// GetOrCreateByName searches by name and optional software type, creating if not found
func (s *Service) GetOrCreateByName(ctx context.Context, name, softwareType string, body any) op.Result[jsonmodels.Software] {
	return op.Result[jsonmodels.Software]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Software] {
		query := model.NewInventoryQuery().
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", name).
			AddFilterEqStr("softwareType", softwareType).
			AddOrderBy("name").
			AddOrderBy("creationTime").
			Build()
		return s.getOrCreateWithQuery(execCtx, body, query)
	}).WithMeta("operation", "getOrCreateByName").
		ExecuteOrDefer(ctx)
}

// GetOrCreateWith provides a generic query-based lookup
// Example queries:
//   - "name eq 'MySoftware' and softwareType eq 'application'"
//   - "name eq 'MySoftware'"
func (s *Service) GetOrCreateWith(ctx context.Context, query string, body any) op.Result[jsonmodels.Software] {
	return op.Result[jsonmodels.Software]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Software] {
		query_ := model.NewInventoryQuery().
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterPart(query).
			AddOrderBy("name").
			AddOrderBy("creationTime").
			Build()
		return s.getOrCreateWithQuery(execCtx, body, query_)
	}).WithMeta("operation", "getOrCreateWith").
		ExecuteOrDefer(ctx)
}

// getOrCreateWithQuery is the internal implementation
func (s *Service) getOrCreateWithQuery(ctx context.Context, body any, query string) op.Result[jsonmodels.Software] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.Software], bool) {
		searchOpts := ListOptions{}
		searchOpts.PageSize = 1
		// Build query with the search criteria
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.Software]{}, false
		}

		// Check if any items were found
		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewSoftware(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		// Not found
		return op.Result[jsonmodels.Software]{}, false
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.Software] {
		createResult := s.Create(ctx, body)
		if createResult.Err != nil {
			return createResult
		}
		return createResult
	}

	// Execute get-or-create pattern
	return op.GetOrCreateR(ctx, finder, creator)
}

// UpsertByName searches by name and optional software type, updating if found or creating if not found
// This ensures metadata stays up-to-date on subsequent calls
func (s *Service) UpsertByName(ctx context.Context, name, softwareType string, body any) op.Result[jsonmodels.Software] {
	return op.Result[jsonmodels.Software]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Software] {
		query := model.NewInventoryQuery().
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", name).
			AddFilterEqStr("softwareType", softwareType).
			AddOrderBy("name").
			AddOrderBy("creationTime").
			Build()
		return s.upsertWithQuery(execCtx, query, body)
	}).WithMeta("operation", "upsertByName").
		ExecuteOrDefer(ctx)
}

// UpsertWith provides a generic query-based upsert
// Updates existing item if found, creates if not found
// Example queries:
//   - "name eq 'MySoftware' and softwareType eq 'application'"
//   - "name eq 'MySoftware' and softwareType eq 'apt' and deviceType eq 'arm64'"
func (s *Service) UpsertWith(ctx context.Context, query string, body any) op.Result[jsonmodels.Software] {
	return op.Result[jsonmodels.Software]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.Software] {
		query_ := model.NewInventoryQuery().
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterPart(query).
			AddOrderBy("name").
			AddOrderBy("creationTime").
			Build()
		return s.upsertWithQuery(execCtx, query_, body)
	}).WithMeta("operation", "upsertWith").
		ExecuteOrDefer(ctx)
}

// upsertWithQuery is the internal implementation for upsert
func (s *Service) upsertWithQuery(ctx context.Context, query string, body any) op.Result[jsonmodels.Software] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.Software], bool) {
		// Build query with the search criteria
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.Software]{}, false
		}

		// Check if any items were found
		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewSoftware(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		// Not found
		return op.Result[jsonmodels.Software]{}, false
	}

	// Define updater function
	updater := func(ctx context.Context, existing op.Result[jsonmodels.Software]) op.Result[jsonmodels.Software] {
		// Update the existing software item with new data
		updateResult := s.Update(ctx, existing.Data.ID(), body)
		if updateResult.Err != nil {
			return updateResult
		}
		return updateResult
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.Software] {
		createResult := s.Create(ctx, body)
		if createResult.Err != nil {
			return createResult
		}
		return createResult
	}

	// Execute upsert pattern
	return op.UpsertR(ctx, finder, updater, creator)
}

// Builder methods for backwards compatibility and flexibility

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParam("forceCascade", fmt.Sprintf("%v", !opt.SkipCascade)).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}
