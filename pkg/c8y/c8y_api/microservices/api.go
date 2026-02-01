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
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMicroservice)
}

// ListAll returns an iterator for all microservices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MicroserviceIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return pagination.Paginate(ctx, func(page int) op.Result[jsonmodels.Microservice] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, jsonmodels.NewMicroservice, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	return s.applicationAPI.ListB(opt.options())
}

// Get an application
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID), jsonmodels.NewMicroservice)
}

func (s *Service) GetB(ID string) *core.TryRequest {
	return s.applicationAPI.GetB(ID)
}

// Create a microservice
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewMicroservice)
}

func (s *Service) CreateB(body any) *core.TryRequest {
	return s.applicationAPI.CreateB(body)
}

// Update a microservice
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, body), jsonmodels.NewMicroservice)
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	return s.applicationAPI.UpdateB(ID, body)
}

type DeleteOptions = applications.DeleteOptions

// Delete a microservice
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.DeleteB(ID, opt), jsonmodels.NewMicroservice)
}

func (s *Service) DeleteB(ID string, opt DeleteOptions) *core.TryRequest {
	return s.applicationAPI.DeleteB(ID, opt)
}

// Subscribe a microservice to a tenant
// TODO: Should 409 errors be ignored? Or should another function be created to allow 409s to be ignored
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfURL string) op.Result[jsonmodels.Microservice] {
	result := core.ExecuteReturnResult(ctx, s.SubscribeB(tenantID, selfURL), func(b []byte) jsonmodels.Microservice {
		// Extract application from MicroserviceReference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewMicroservice([]byte(doc.Get("application").Raw))
	})
	return result
}

func (s *Service) SubscribeB(tenantID string, selfURL string) *core.TryRequest {
	return s.applicationAPI.SubscribeB(tenantID, selfURL)
}

// Unsubscribe a microservice from a tenant
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, ID string) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.UnsubscribeB(tenantID, ID), jsonmodels.NewMicroservice)
}

func (s *Service) UnsubscribeB(tenantID string, ID string) *core.TryRequest {
	return s.applicationAPI.UnsubscribeB(tenantID, ID)
}

type UploadFileOptions = applications.UploadFileOptions

// Upload a new microservice binary
func (s *Service) Upload(ctx context.Context, ID string, opt UploadFileOptions) op.Result[jsonmodels.MicroserviceBinary] {
	return core.ExecuteReturnResult(ctx, s.UploadB(ID, opt), jsonmodels.NewMicroserviceBinary)
}

func (s *Service) UploadB(ID string, opt UploadFileOptions) *core.TryRequest {
	return s.applicationAPI.UploadB(ID, opt)
}
