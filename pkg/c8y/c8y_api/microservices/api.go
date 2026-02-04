package microservices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/applications"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/microservices/bootstrapuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/microservices/currentmicroservice"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

const ResultProperty = "applications"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct {
	core.Service
	applicationAPI      applications.Service
	BootstrapUser       bootstrapuser.Service
	CurrentMicroservice currentmicroservice.Service
}

func NewService(common *core.Service) *Service {
	return &Service{
		Service:             *common,
		applicationAPI:      *applications.NewService(common),
		BootstrapUser:       *bootstrapuser.NewService(common),
		CurrentMicroservice: *currentmicroservice.NewService(common),
	}
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
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMicroservice)
}

// ListAll returns an iterator for all microservices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MicroserviceIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.Microservice] {
		return s.List(ctx, opts)
	}, jsonmodels.NewMicroservice)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	// Build request directly since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt.options())).
		SetURL(applications.ApiApplications)
	return core.NewTryRequest(s.Client, req, applications.ResultProperty)
}

// Get an application
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewMicroservice)
}

func (s *Service) getB(ID string) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam("id", ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplication)
	return core.NewTryRequest(s.Client, req, "")
}

// Create a microservice
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewMicroservice)
}

func (s *Service) createB(body any) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplications)
	return core.NewTryRequest(s.Client, req, "")
}

// Update a microservice
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewMicroservice)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(applications.ApiApplication)
	return core.NewTryRequest(s.Client, req, "")
}

type DeleteOptions = applications.DeleteOptions

// Delete a microservice
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.deleteB(ID, opt), jsonmodels.NewMicroservice)
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("id", ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(applications.ApiApplication)
	return core.NewTryRequest(s.Client, req, "")
}

// Subscribe a microservice to a tenant
// TODO: Should 409 errors be ignored? Or should another function be created to allow 409s to be ignored
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfURL string) op.Result[jsonmodels.Microservice] {
	result := core.Execute(ctx, s.subscribeB(tenantID, selfURL), func(b []byte) jsonmodels.Microservice {
		// Extract application from MicroserviceReference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewMicroservice([]byte(doc.Get("application").Raw))
	})
	return result
}

func (s *Service) subscribeB(tenantID string, selfURL string) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("tenantId", tenantID).
		SetBody(map[string]any{"application": map[string]any{"self": selfURL}}).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/tenant/tenants/{tenantId}/applications")
	return core.NewTryRequest(s.Client, req, "")
}

// Unsubscribe a microservice from a tenant
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, ID string) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.unsubscribeB(tenantID, ID), jsonmodels.NewMicroservice)
}

func (s *Service) unsubscribeB(tenantID string, ID string) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam("tenantId", tenantID).
		SetPathParam("id", ID).
		SetURL("/tenant/tenants/{tenantId}/applications/{id}")
	return core.NewTryRequest(s.Client, req, "")
}

type UploadFileOptions = applications.UploadFileOptions

// Upload a new microservice binary
func (s *Service) Upload(ctx context.Context, ID string, opt UploadFileOptions) op.Result[jsonmodels.MicroserviceBinary] {
	return core.Execute(ctx, s.uploadB(ID, opt), jsonmodels.NewMicroserviceBinary)
}

func (s *Service) uploadB(ID string, opt UploadFileOptions) *core.TryRequest {
	// Rebuild request since applications B methods are now private
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam("id", ID).
		SetFileReader("file", opt.Name, opt.Reader).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL("/application/applications/{id}/binaries")
	return core.NewTryRequest(s.Client, req, "")
}
