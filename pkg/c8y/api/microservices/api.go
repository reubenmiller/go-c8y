package microservices

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices/bootstrapuser"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices/currentmicroservice"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices/loggers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
	"resty.dev/v3"
)

const ResultProperty = "applications"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct {
	svc            httpcore.Service
	applicationAPI *applications.Service
	BootstrapUser  bootstrapuser.Service
	// CurrentMicroservice is the current-microservice sub-resource.
	//
	// Deprecated: use Client.Microservices.Current.
	CurrentMicroservice currentmicroservice.Service
	Loggers             loggers.Service

	// Current is the current-microservice sub-resource (/application/currentApplication
	// as seen by a microservice). It is the canonical name, matching
	// Tenants.Current, Applications.Current and Users.Current, and points at
	// the same instance as CurrentMicroservice.
	Current *currentmicroservice.Service

	lookupByName        func(ctx context.Context, name string) (string, map[string]any, error)
	lookupByContextPath func(ctx context.Context, contextPath string) (string, map[string]any, error)
	customResolvers     map[string]source.Resolver
}

func NewService(common *httpcore.Service) *Service {
	return NewServiceWithApplications(common, applications.NewService(common))
}

// NewServiceWithApplications creates the service using an existing applications
// service, so that resolvers and lookups registered on it (e.g.
// Client.Applications) are shared rather than copied.
func NewServiceWithApplications(common *httpcore.Service, apps *applications.Service) *Service {
	service := &Service{
		svc:                 *common,
		applicationAPI:      apps,
		BootstrapUser:       *bootstrapuser.NewService(common),
		CurrentMicroservice: *currentmicroservice.NewService(common),
		Loggers:             *loggers.NewService(common),
		customResolvers:     make(map[string]source.Resolver),
	}

	service.Current = &service.CurrentMicroservice

	// Setup lookup function for name-based resolution
	service.lookupByName = func(ctx context.Context, name string) (string, map[string]any, error) {
		opts := ListOptions{
			PaginationOptions: pagination.PaginationOptions{
				MaxItems: 4000,
			},
		}

		it := service.ListAll(ctx, opts)
		if it.Err() != nil {
			return "", nil, it.Err()
		}

		// Client-side filtering with wildcard support
		for item, err := range it.Items() {
			if err != nil {
				return "", nil, err
			}
			if found, _ := matcher.MatchWithWildcards(item.Name(), name); found {
				return item.ID(), map[string]any{
					"id":          item.ID(),
					"name":        item.Name(),
					"contextPath": item.ContextPath(),
				}, nil
			}
		}

		return "", nil, core.NewNameNotFoundError("microservice", name, "microservice not found with name: %s", name)
	}

	// Setup lookup function for contextPath-based resolution
	service.lookupByContextPath = func(ctx context.Context, contextPath string) (string, map[string]any, error) {
		opts := ListOptions{
			PaginationOptions: pagination.PaginationOptions{
				MaxItems: 4000,
			},
		}

		it := service.ListAll(ctx, opts)
		if it.Err() != nil {
			return "", nil, it.Err()
		}

		// Client-side filtering with wildcard support
		for item, err := range it.Items() {
			if err != nil {
				return "", nil, err
			}
			if found, _ := matcher.MatchWithWildcards(item.ContextPath(), contextPath); found {
				return item.ID(), map[string]any{
					"id":          item.ID(),
					"name":        item.Name(),
					"contextPath": item.ContextPath(),
				}, nil
			}
		}

		return "", nil, core.ErrNotFound("microservice not found with contextPath: %s", contextPath)
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

	// The ID of a user that has access to the applications
	User string `url:"user,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

func (lo *ListOptions) options() applications.ListOptions {
	return applications.ListOptions{
		Type:              applications.TypeMicroservice,
		Name:              lo.Name,
		Owner:             lo.Owner,
		ProvidedFor:       lo.ProvidedFor,
		Subscriber:        lo.Subscriber,
		Tenant:            lo.Tenant,
		User:              lo.User,
		PaginationOptions: lo.PaginationOptions,
	}
}

func ByName(v string) func(model.Microservice) bool {
	return func(m model.Microservice) bool {
		return m.Name == v
	}
}
func First(m model.Microservice) bool {
	return true
}

// MicroserviceIterator provides iteration over microservices
type MicroserviceIterator = pagination.Iterator[jsonmodels.Microservice]

func (s *Service) FindFirst(ctx context.Context, opt ListOptions) (op.Result[jsonmodels.Microservice], bool) {
	opt.MaxItems = 1
	iterator := s.ListAll(ctx, opt)
	if iterator.Err() != nil {
		return op.Failed[jsonmodels.Microservice](iterator.Err(), false), false
	}
	return op.First(iterator.Items())
}

// List all microservices on your tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Microservice] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMicroservice)
}

// ListAll returns an iterator for all microservices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MicroserviceIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Microservice] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewMicroservice,
	)
}

// ByID returns a direct ID reference (no lookup needed).
// Returns: "12345"
func (s *Service) ByID(id string) string {
	return id
}

// ByName creates a name-based reference string for microservice lookup.
// Returns: "name:serviceName"
// The actual lookup will be performed when this string is resolved via ResolveID
// Example: service.Get(ctx, service.ByName("my-microservice"))
func (s *Service) ByName(name string) string {
	return fmt.Sprintf("name:%s", name)
}

// ByContextPath creates a contextPath-based reference string for microservice lookup.
// Returns: "contextPath:/path"
// The actual lookup will be performed when this string is resolved via ResolveID
// Example: service.Get(ctx, service.ByContextPath("/my-microservice"))
func (s *Service) ByContextPath(contextPath string) string {
	return fmt.Sprintf("contextPath:%s", contextPath)
}

// ResolveID resolves a microservice ID string that may contain a resolver scheme.
// If meta is not nil, it will be populated with metadata about the resolution.
// Examples:
//   - "12345" -> "12345" (plain ID, meta: {"source": "direct-id"})
//   - "name:my-microservice" -> "<id>" (meta: {"name": "my-microservice", ...})
//   - "contextPath:/my-microservice" -> "<id>" (meta: {"contextPath": "/my-microservice", ...})
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("microservice", id, resolveErr) }()
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	// Build request directly since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt.options())).
		SetURL(applications.ApiApplications)
	return httpcore.NewTryRequest(s.svc.Client, req, applications.ResultProperty)
}

// Get a microservice by ID or resolver string
// Examples:
//   - Get(ctx, "12345") - direct ID
//   - Get(ctx, "name:my-microservice") - lookup by name
//   - Get(ctx, "contextPath:/my-microservice") - lookup by context path
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.Microservice] {
	// Resolve ID (supports "name:serviceName", "contextPath:/path", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Microservice](err, false)
	}

	return httpcore.Execute(ctx, s.getB(resolvedID), jsonmodels.NewMicroservice, meta)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam("id", ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Create a microservice
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Microservice] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewMicroservice, nil)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplications)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Update a microservice by ID or resolver string
// Examples:
//   - Update(ctx, "12345", body) - direct ID
//   - Update(ctx, "name:my-microservice", body) - lookup by name
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.Microservice] {
	// Resolve ID (supports "name:serviceName", "contextPath:/path", etc.)
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.Microservice](err, false)
	}

	return httpcore.Execute(ctx, s.updateB(resolvedID, body), jsonmodels.NewMicroservice, meta)
}

func (s *Service) updateB(ID string, body any) *httpcore.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", ID).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

type DeleteOptions = applications.DeleteOptions

// Delete a microservice by ID or resolver string.
// A 404 response (already deleted) is treated as skipped (StatusSkipped, Idempotent: true).
// Examples:
//   - Delete(ctx, "12345", opts) - direct ID
//   - Delete(ctx, "name:my-microservice", opts) - lookup by name
func (s *Service) Delete(ctx context.Context, id string, opt DeleteOptions) op.Result[core.NoContent] {
	// Resolve ID (supports "name:serviceName", "contextPath:/path", etc.)
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
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(applications.ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Subscribe a microservice to a tenant.
// A 409 response (already subscribed) is treated as a duplicate (StatusDuplicate, Idempotent: true).
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfURL string) op.Result[jsonmodels.Microservice] {
	return httpcore.Execute(ctx, s.subscribeB(tenantID, selfURL), func(b []byte) jsonmodels.Microservice {
		// Extract application from MicroserviceReference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewMicroservice([]byte(doc.Get("application").Raw))
	}).IgnoreConflict()
}

func (s *Service) subscribeB(tenantID string, selfURL string) *httpcore.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("tenantId", tenantID).
		SetBody(map[string]any{"application": map[string]any{"self": selfURL}}).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/tenant/tenants/{tenantId}/applications")
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Unsubscribe a microservice from a tenant by ID or resolver string.
// A 404 response (not subscribed) is treated as skipped (StatusSkipped, Idempotent: true).
// Examples:
//   - Unsubscribe(ctx, tenantID, "12345") - direct ID
//   - Unsubscribe(ctx, tenantID, "name:my-microservice") - lookup by name
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, id string) op.Result[core.NoContent] {
	// Resolve ID (supports "name:serviceName", "contextPath:/path", etc.)
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
	// Rebuild request since applications B methods are now private
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetPathParam("id", ID).
		SetURL("/tenant/tenants/{tenantId}/applications/{id}")
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Enable (subscribe) a microservice for a tenant by POSTing an application
// reference body to /tenant/tenants/{tenant}/applications. The body identifies
// the microservice by id or self link, e.g. {"application":{"id":"<id>"}}; the
// CLI resolves the microservice reference and sets application.id before calling
// this. Returns the created application reference. Unlike Subscribe (which
// plucks the application for the create/update flow), this returns the reference
// as-is to match the v1 enable command's applicationReference output.
func (s *Service) Enable(ctx context.Context, tenantID string, body any) op.Result[jsonmodels.ApplicationReference] {
	return httpcore.Execute(ctx, s.enableB(tenantID, body), jsonmodels.NewApplicationReference, nil)
}

func (s *Service) enableB(tenantID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/tenant/tenants/{tenantId}/applications")
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GetStatus returns the managed object(s) holding a microservice's status,
// which Cumulocity stores in the inventory under the type c8y_Application_<id>.
// The id must already be resolved (the CLI resolves a name/contextPath
// reference first). The collection is plucked into managedObjects items.
func (s *Service) GetStatus(ctx context.Context, id string) op.Result[jsonmodels.ManagedObject] {
	return httpcore.ExecuteCollection(ctx, s.getStatusB(id), "managedObjects", types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

func (s *Service) getStatusB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParam("type", "c8y_Application_"+id).
		SetURL("/inventory/managedObjects")
	return httpcore.NewTryRequest(s.svc.Client, req, "managedObjects")
}

type UploadFileOptions = applications.UploadFileOptions

// Upload a new microservice binary by ID or resolver string
// Examples:
//   - Upload(ctx, "12345", opts) - direct ID
//   - Upload(ctx, "name:my-microservice", opts) - lookup by name
func (s *Service) Upload(ctx context.Context, id string, opt UploadFileOptions) op.Result[jsonmodels.MicroserviceBinary] {
	// Resolve ID (supports "name:serviceName", "contextPath:/path", etc.)
	resolutionCtx := ctxhelpers.ResolutionContext(ctx)

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, id, meta)
	if err != nil {
		return op.Failed[jsonmodels.MicroserviceBinary](err, false)
	}

	return httpcore.Execute(ctx, s.uploadB(resolvedID, opt), jsonmodels.NewMicroserviceBinary, meta)
}

func (s *Service) uploadB(ID string, opt UploadFileOptions) *httpcore.TryRequest {
	// Mirror applications.uploadB: build the multipart "file" (+ optional object
	// metadata) from the UploadFileOptions so a FilePath-only caller (the CLI
	// createBinary command) streams the file. The previous SetFileReader form
	// required an explicit Reader and silently sent nothing for FilePath callers.
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("id", ID).
		SetMultipartFields(httpcore.NewMultiPartFile(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/application/applications/{id}/binaries")
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}
