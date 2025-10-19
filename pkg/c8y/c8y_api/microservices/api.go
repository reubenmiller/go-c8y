package microservices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/applications"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
)

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct {
	core.Service
	applicationAPI applications.Service
}

func NewService(common *core.Service) *Service {
	return &Service{
		Service:        *common,
		applicationAPI: *applications.NewService(common),
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

func (s *Service) FindFirst(ctx context.Context, opt ListOptions) (*model.Microservice, bool, error) {
	return pagination.ForEachWhere(ctx, s.ListB(opt), First)
}

// List all microservices on your tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.MicroserviceCollection, error) {
	return core.ExecuteResultOnly[model.MicroserviceCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	return s.applicationAPI.ListB(opt.options())
}

// Get an application
func (s *Service) Get(ctx context.Context, ID string) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.GetB(ID))
}

func (s *Service) GetB(ID string) *core.TryRequest {
	return s.applicationAPI.GetB(ID)
}

// Create a microservice
func (s *Service) Create(ctx context.Context, body any) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	return s.applicationAPI.CreateB(body)
}

// Update a microservice
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	return s.applicationAPI.UpdateB(ID, body)
}

type DeleteOptions = applications.DeleteOptions

// Delete a microservice
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(ID, opt))
}

func (s *Service) DeleteB(ID string, opt DeleteOptions) *core.TryRequest {
	return s.applicationAPI.DeleteB(ID, opt)
}

// Subscribe a microservice to a tenant
func (s *Service) Subscribe(ctx context.Context, tenantID string, selfURL string) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.SubscribeB(tenantID, selfURL))
}

func (s *Service) SubscribeB(tenantID string, selfURL string) *core.TryRequest {
	return s.applicationAPI.SubscribeB(tenantID, selfURL)
}

// Unsubscribe a microservice from a tenant
func (s *Service) Unsubscribe(ctx context.Context, tenantID string, ID string) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.UnsubscribeB(tenantID, ID))
}

func (s *Service) UnsubscribeB(tenantID string, ID string) *core.TryRequest {
	return s.applicationAPI.UnsubscribeB(tenantID, ID)
}

type UploadFileOptions = applications.UploadFileOptions

// Upload a new microservice binary
func (s *Service) Upload(ctx context.Context, ID string, opt UploadFileOptions) (*model.Microservice, error) {
	return core.ExecuteResultOnly[model.Microservice](ctx, s.UploadB(ID, opt))
}

func (s *Service) UploadB(ID string, opt UploadFileOptions) *core.TryRequest {
	return s.applicationAPI.UploadB(ID, opt)
}
