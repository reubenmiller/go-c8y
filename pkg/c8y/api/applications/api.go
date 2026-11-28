package applications

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications/currentapplication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications/versions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	apiApplications          = "/application/applications"
	apiApplication           = "/application/applications/{id}"
	apiApplicationBinaries   = "/application/applications/{id}/binaries"
	apiApplicationBinary     = "/application/applications/{id}/binaries/{binaryId}"
	apiApplicationClone      = "/application/applications/{id}/clone"
	apiApplicationByName     = "/application/applicationsByName/{name}"
	apiApplicationByTenantID = "/application/applicationsByTenant/{tenantId}"
	apiApplicationByOwner    = "/application/applicationsByOwner/{tenantId}"
	apiApplicationByUser     = "/application/applicationsByUser/{username}"
	apiTenantApplications    = "/tenant/tenants/{tenantId}/applications"
	apiTenantApplication     = "/tenant/tenants/{tenantId}/applications/{id}"
)

var (
	TypeMicroservice = "MICROSERVICE"
	TypeHosted       = "HOSTED"
)

var paramID = "id"
var paramName = "name"
var paramUsername = "username"

const resultProperty = "applications"

// Service to manage applications
type Service struct {
	svc httpcore.Service
	// Current is the current-application sub-resource (/application/currentApplication)
	Current *currentapplication.Service

	// Versions is the application versions sub-resource.
	Versions *versions.Service

	// Resolver lookup function
	lookupByName func(ctx context.Context, name, appType string) (string, map[string]any, error)
	resolvers    *source.Registry
}

func NewService(common *httpcore.Service) *Service {
	service := &Service{
		svc:      *common,
		Current:  currentapplication.NewService(common),
		Versions: versions.NewService(common),
	}
	service.resolvers = service.newResolvers()

	// Setup lookup function for name-based resolution
	service.lookupByName = func(ctx context.Context, name, appType string) (string, map[string]any, error) {
		opts := ListOptions{
			Type: appType,
			PaginationOptions: pagination.PaginationOptions{
				MaxItems: 2000,
			},
		}

		app, found, err := pagination.FindWildcard(service.ListAll(ctx, opts), name, func(a jsonmodels.Application) []string { return []string{a.Name()} })
		if err != nil {
			return "", nil, err
		}
		if found {
			return app.ID(), map[string]any{
				"id":   app.ID(),
				"name": app.Name(),
				"type": app.Type(),
			}, nil
		}

		if appType != "" {
			return "", nil, core.NewNameNotFoundError("application", name, "application not found with name: %s, type: %s", name, appType)
		}
		return "", nil, core.NewNameNotFoundError("application", name, "application not found with name: %s", name)
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

	// Application access level for other tenants (e.g. MARKET, PRIVATE)
	Availability string `url:"availability,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// ApplicationIterator provides iteration over applications
type ApplicationIterator = pagination.Iterator[jsonmodels.Application]

// List all applications on your tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplications)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type ListByNameOptions struct {
	// The name of the application
	Name string

	// Pagination options
	pagination.PaginationOptions
}

// ListByName lists the applications with the given name
// (GET /application/applicationsByName/{name}). Use [Service.List] to combine
// the name with other filters.
func (s *Service) ListByName(ctx context.Context, opt ListByNameOptions) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.listByNameB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByNameB(opt ListByNameOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramName, opt.Name).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplicationByName)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type ListByTenantOptions struct {
	// Unique identifier of a Cumulocity tenant
	TenantID string

	// Pagination options
	pagination.PaginationOptions
}

// ListByTenant lists the applications that a tenant owns or is subscribed to
// (GET /application/applicationsByTenant/{tenantId}).
func (s *Service) ListByTenant(ctx context.Context, opt ListByTenantOptions) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.listByTenantB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByTenantB(opt ListByTenantOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplicationByTenantID)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type ListByOwnerOptions struct {
	// Unique identifier of a Cumulocity tenant
	TenantID string

	// Pagination options
	pagination.PaginationOptions
}

// Retrieve all applications owned by a particular tenant (by a given tenant ID)
func (s *Service) ListByOwner(ctx context.Context, opt ListByOwnerOptions) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.listByOwnerB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByOwnerB(opt ListByOwnerOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplicationByOwner)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type ListByUserOptions struct {
	// Unique identifier of a Cumulocity tenant
	Username string

	// Pagination options
	pagination.PaginationOptions
}

// Retrieve all applications for a particular user (by a given username)
func (s *Service) ListByUser(ctx context.Context, opt ListByUserOptions) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.listByUserB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplication)
}

func (s *Service) listByUserB(opt ListByUserOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramUsername, opt.Username).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplicationByUser)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
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
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return httpcore.Execute(ctx, s.getB(resolvedID), jsonmodels.NewApplication, meta)
}

func (s *Service) getB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetURL(apiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create an application
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewApplication, nil)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplications)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update an application by ID or resolver string
// Examples:
//   - Update(ctx, "12345", body) - direct ID
//   - Update(ctx, "name:cockpit", body) - lookup by name
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return httpcore.Execute(ctx, s.updateB(resolvedID, body), jsonmodels.NewApplication, meta)
}

func (s *Service) updateB(ID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(paramID, ID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}

	return httpcore.ExecuteNoContent(ctx, s.deleteB(resolvedID, opt), meta).IgnoreNotFound()
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, ID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return httpcore.Execute(ctx, s.copyB(resolvedID, opt), jsonmodels.NewApplication, meta)
}

func (s *Service) copyB(ID string, opt CopyOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, ID).
		SetURL(apiApplicationClone)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Subscribe an application to a tenant.
// A 409 response (already subscribed) is treated as a duplicate (StatusDuplicate, Idempotent: true).
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfLink string) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.subscribeB(tenantID, selfLink), jsonmodels.NewApplication, nil).IgnoreConflict()
}

func (s *Service) subscribeB(tenantID string, selfURL string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetBody(model.NewApplicationReference(selfURL)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenantApplications)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Unsubscribe an application from a tenant by ID or resolver string
// Examples:
//   - Unsubscribe(ctx, "tenant01", "12345") - direct ID
//   - Unsubscribe(ctx, "tenant01", "name:cockpit") - lookup by name
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, id string) op.Result[core.NoContent] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}

	return httpcore.ExecuteNoContent(ctx, s.unsubscribeB(tenantID, resolvedID), meta).IgnoreNotFound()
}

func (s *Service) unsubscribeB(tenantID string, ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetPathParam(paramID, ID).
		SetURL(apiTenantApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Upload an application binary by ID or resolver string
// Examples:
//   - Upload(ctx, "12345", opts) - direct ID
//   - Upload(ctx, "name:myapp", opts) - lookup by name
func (s *Service) Upload(ctx context.Context, id string, opt UploadFileOptions) op.Result[jsonmodels.Application] {
	// Resolve ID (supports "name:appName", etc.)
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	return httpcore.Execute(ctx, s.uploadB(resolvedID, opt), jsonmodels.NewApplication, meta)
}

func (s *Service) uploadB(ID string, opt UploadFileOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(paramID, ID).
		SetMultipartFields(httpcore.NewMultiPartFile(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplicationBinaries)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListBinaries lists the binary attachments of an application. The id may be a
// plain id or a resolver string ("name:appName"). The attachments are returned
// as a flat (non-paginated) collection under the "attachments" property.
func (s *Service) ListBinaries(ctx context.Context, id string) op.Result[jsonmodels.ApplicationBinary] {
	resolvedID, err := s.ResolveID(ctxhelpers.ResolutionContext(ctx), id, nil)
	if err != nil {
		return op.Failed[jsonmodels.ApplicationBinary](err, false)
	}
	return httpcore.ExecuteCollection(ctx, s.listBinariesB(resolvedID), "attachments", types.ResponseFieldStatistics, jsonmodels.NewApplicationBinary)
}

func (s *Service) listBinariesB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetURL(apiApplicationBinaries)
	return httpcore.NewTryRequest(s.svc.Client, req, "attachments")
}

// DeleteBinary deletes a binary attachment of an application by binary id. The
// application id may be a plain id or a resolver string ("name:appName").
func (s *Service) DeleteBinary(ctx context.Context, id string, binaryID string) op.Result[core.NoContent] {
	resolvedID, err := s.ResolveID(ctxhelpers.ResolutionContext(ctx), id, nil)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	return httpcore.ExecuteNoContent(ctx, s.deleteBinaryB(resolvedID, binaryID)).IgnoreNotFound()
}

func (s *Service) deleteBinaryB(id string, binaryID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, id).
		SetPathParam("binaryId", binaryID).
		SetURL(apiApplicationBinary)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (_ string, resolveErr error) {
	return s.resolvers.Resolve(ctx, id, meta)
}

// RegisterResolver allows registering custom ID resolvers for use with ResolveID
// Example: RegisterResolver("custom", myResolver)
// Then use: ResolveID(ctx, "custom:value")
func (s *Service) RegisterResolver(scheme string, resolver source.Resolver) {
	s.resolvers.Register(scheme, resolver)
}
