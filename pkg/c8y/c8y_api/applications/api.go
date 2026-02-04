package applications

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/matcher"
	"resty.dev/v3"
)

var (
	ApiApplications          = "/application/applications"
	ApiApplication           = "/application/applications/{id}"
	ApiApplicationBinaries   = "/application/applications/{id}/binaries"
	ApiApplicationClone      = "/application/applications/{id}/clone"
	ApiApplicationByName     = "/application/applicationsByName/{name}"
	ApiApplicationByTenantID = "/application/applicationsByTenant/{tenantID}"
	ApiApplicationByOwner    = "/application/applicationsByTenant/{tenantID}"
	ApiApplicationByUser     = "/application/applicationsByUser/{username}"
	ApiTenantApplications    = "/tenant/tenants/{tenantID}/applications"
	ApiTenantApplication     = "/tenant/tenants/{tenantID}/applications/{id}"
)

var (
	TypeMicroservice = "MICROSERVICE"
	TypeHosted       = "HOSTED"
)

var ParamId = "id"
var ParamName = "name"
var ParamTenantID = "tenantID"
var ParamUsername = "username"

const ResultProperty = "applications"

// Service to manage applications
type Service struct {
	core.Service

	// Resolver lookup function
	lookupByName    func(ctx context.Context, name, appType string) (string, map[string]any, error)
	customResolvers map[string]source.Resolver
}

func NewService(common *core.Service) *Service {
	service := &Service{
		Service:         *common,
		customResolvers: make(map[string]source.Resolver),
	}

	// Setup lookup function for name-based resolution
	service.lookupByName = func(ctx context.Context, name, appType string) (string, map[string]any, error) {
		opts := ListOptions{
			Type: appType,
			PaginationOptions: pagination.PaginationOptions{
				MaxItems: 2000,
			},
		}

		it := service.ListAll(ctx, opts)
		if it.Err() != nil {
			return "", nil, it.Err()
		}

		// Client-side filtering with wildcard support
		for item := range it.Items() {
			if found, _ := matcher.MatchWithWildcards(item.Name(), name); found {
				return item.ID(), map[string]any{
					"id":   item.ID(),
					"name": item.Name(),
					"type": item.Type(),
				}, nil
			}
		}

		if appType != "" {
			return "", nil, fmt.Errorf("application not found with name: %s, type: %s", name, appType)
		}
		return "", nil, fmt.Errorf("application not found with name: %s", name)
	}

	return service
}

// ListOptions filter options
type ListOptions struct {
	// The name of the application
	Name string `url:"name,omitempty"`

	// The ID of the tenant that owns the applications
	Owner string `url:"owner,omitempty"`

	// The ID of a tenant that is subscribed to the applications but doesn't own them
	ProvidedFor string `url:"providedFor,omitempty"`

	// The ID of a tenant that is subscribed to the applications
	Subscriber string `url:"subscriber,omitempty"`

	// The ID of a tenant that either owns the application or is subscribed to the applications
	Tenant string `url:"tenant,omitempty"`

	// The type of the application. It is possible to use multiple values separated by a comma. For example, EXTERNAL,HOSTED will return only applications with type EXTERNAL or HOSTED
	Type string `url:"type,omitempty"`

	// The ID of a user that has access to the applications
	User string `url:"user,omitempty"`

	// When set to true, the returned result contains applications with an applicationVersions
	// field that is not empty. When set to false, the result will contain applications with an
	// empty applicationVersions field
	HasVersions string `url:"hasVersions,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// ApplicationIterator provides iteration over applications
type ApplicationIterator = pagination.Iterator[jsonmodels.Application]

// List all applications on your tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

// ListAll returns an iterator for all applications
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ApplicationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Application] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewApplication,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiApplications)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type ListByNameOptions struct {
	// The name of the application
	Name string

	// Pagination options
	pagination.PaginationOptions
}

// List applications by name
func (s *Service) ListByName(ctx context.Context, opt ListByNameOptions) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.listByNameB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByNameB(opt ListByNameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiApplicationByName)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type ListByTenantOptions struct {
	// Unique identifier of a Cumulocity tenant
	TenantID string

	// Pagination options
	pagination.PaginationOptions
}

// List applications by name
func (s *Service) ListByTenant(ctx context.Context, opt ListByTenantOptions) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.listByTenantB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByTenantB(opt ListByTenantOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenantID, opt.TenantID).
		SetURL(ApiApplicationByTenantID)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type ListByOwnerOptions struct {
	// Unique identifier of a Cumulocity tenant
	TenantID string

	// Pagination options
	pagination.PaginationOptions
}

// Retrieve all applications owned by a particular tenant (by a given tenant ID)
func (s *Service) ListByOwner(ctx context.Context, opt ListByOwnerOptions) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.listByOwnerB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByOwnerB(opt ListByOwnerOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenantID, opt.TenantID).
		SetURL(ApiApplicationByTenantID)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type ListByUserOptions struct {
	// Unique identifier of a Cumulocity tenant
	Username string

	// Pagination options
	pagination.PaginationOptions
}

// Retrieve all applications for a particular user (by a given username)
func (s *Service) ListByUser(ctx context.Context, opt ListByUserOptions) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.listByUserB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByUserB(opt ListByUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamUsername, opt.Username).
		SetURL(ApiApplicationByUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an application by ID or resolver string
// Examples:
//   - Get(ctx, "12345") - direct ID
//   - Get(ctx, "name:cockpit") - lookup by name
//   - Get(ctx, "name:cockpit:HOSTED") - lookup by name and type
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", "name:appName:HOSTED", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Use background context for resolution so it doesn't inherit the deferred flag
		// This allows lookups (like ListAll) to actually execute
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return core.Execute(ctx, s.getB(resolvedID), jsonmodels.NewApplication, meta)
}

func (s *Service) getB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, id).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// Create an application
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Application] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewApplication, nil)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplications)
	return core.NewTryRequest(s.Client, req)
}

// Update an application by ID or resolver string
// Examples:
//   - Update(ctx, "12345", body) - direct ID
//   - Update(ctx, "name:cockpit", body) - lookup by name
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return core.Execute(ctx, s.updateB(resolvedID, body), jsonmodels.NewApplication, meta)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

type DeleteOptions struct {
	// Force deletion by unsubscribing all tenants from the application first and then deleting the application itself
	Force bool `url:"force,omitempty"`
}

// Delete an application by ID or resolver string
// Examples:
//   - Delete(ctx, "12345", opts) - direct ID
//   - Delete(ctx, "name:cockpit", opts) - lookup by name
func (s *Service) Delete(ctx context.Context, id string, opt DeleteOptions) op.Result[core.NoContent] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}

	return core.ExecuteNoContent(ctx, s.deleteB(resolvedID, opt), meta)
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

type CopyOptions struct {
	// The version field of the application version
	Version string `url:"version,omitempty"`

	// The tag of the application version
	Tag string `url:"tag,omitempty"`
}

// Copy an application by ID or resolver string
// Examples:
//   - Copy(ctx, "12345", opts) - direct ID
//   - Copy(ctx, "name:cockpit", opts) - lookup by name
func (s *Service) Copy(ctx context.Context, id string, opt CopyOptions) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return core.Execute(ctx, s.copyB(resolvedID, opt), jsonmodels.NewApplication, meta)
}

func (s *Service) copyB(ID string, opt CopyOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, ID).
		SetURL(ApiApplicationClone)
	return core.NewTryRequest(s.Client, req)
}

// Subscribe an application to a tenant
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfLink string) op.Result[jsonmodels.Application] {
	return core.Execute(ctx, s.subscribeB(tenantID, selfLink), jsonmodels.NewApplication, nil)
}

func (s *Service) subscribeB(tenantID string, selfURL string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenantID, tenantID).
		SetBody(model.NewApplicationReference(selfURL)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantApplications)
	return core.NewTryRequest(s.Client, req)
}

// Unsubscribe an application from a tenant by ID or resolver string
// Examples:
//   - Unsubscribe(ctx, "tenant01", "12345") - direct ID
//   - Unsubscribe(ctx, "tenant01", "name:cockpit") - lookup by name
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, id string) op.Result[core.NoContent] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}

	return core.ExecuteNoContent(ctx, s.unsubscribeB(tenantID, resolvedID), meta)
}

func (s *Service) unsubscribeB(tenantID string, ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantID, tenantID).
		SetPathParam(ParamId, ID).
		SetURL(ApiTenantApplication)
	return core.NewTryRequest(s.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Upload an application binary by ID or resolver string
// Examples:
//   - Upload(ctx, "12345", opts) - direct ID
//   - Upload(ctx, "name:myapp", opts) - lookup by name
func (s *Service) Upload(ctx context.Context, id string, opt UploadFileOptions) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return core.Execute(ctx, s.uploadB(resolvedID, opt), jsonmodels.NewApplication, meta)
}

func (s *Service) uploadB(ID string, opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, ID).
		SetMultipartFields(core.NewMultiPartFile(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplicationBinaries)
	return core.NewTryRequest(s.Client, req)
}

// Application Resolution Convenience Methods
// These methods provide a more discoverable way to create source resolvers
// for applications, while still returning the generic source.Resolver interface.

// ByID creates a resolver for an application by its direct ID.
// Returns a source.Resolver that can be used with any API that accepts source resolution.
// ByID creates a direct ID reference string (no lookup needed).
// Returns: "12345"
func (s *Service) ByID(id string) string {
	return id
}

// ByName creates a name-based reference string for application lookup.
// Returns: "name:appName" or "name:appName:type" if type is specified
// The actual lookup will be performed when this string is resolved via ResolveID
func (s *Service) ByName(name string, appType string) string {
	if appType != "" {
		return fmt.Sprintf("name:%s:%s", name, appType)
	}
	return fmt.Sprintf("name:%s", name)
}

// ResolveID resolves an application ID string that may contain a resolver scheme.
// If meta is not nil, it will be populated with metadata about the resolution.
// Examples:
//   - "12345" -> "12345" (plain ID, meta: {"source": "direct-id"})
//   - "name:cockpit" -> "<id>" (meta: {"name": "cockpit", "type": "...", ...})
//   - "name:cockpit:HOSTED" -> "<id>" (meta: {"name": "cockpit", "type": "HOSTED", ...})
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (string, error) {
	resolver, err := s.parseResolver(id)
	if err != nil {
		return "", err
	}
	result, err := resolver.ResolveID(ctx)
	if err != nil {
		return "", err
	}
	if meta != nil {
		for k, v := range result.Meta {
			meta[k] = v
		}
	}
	return result.ID, nil
}

// RegisterResolver allows registering custom ID resolvers for use with ResolveID
// Example: RegisterResolver("custom", myResolver)
// Then use: ResolveID(ctx, "custom:value")
func (s *Service) RegisterResolver(scheme string, resolver source.Resolver) {
	s.customResolvers[scheme] = resolver
}
