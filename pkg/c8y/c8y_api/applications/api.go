package applications

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiApplications = "/application/applications"
var ApiApplication = "/application/applications/{id}"
var ApiApplicationClone = "/application/applications/{id}/clone"
var ApiApplicationByName = "/application/applicationsByName/{name}"
var ApiApplicationByTenantID = "/application/applicationsByTenant/{tenantID}"
var ApiApplicationByOwner = "/application/applicationsByTenant/{tenantID}"
var ApiApplicationByUser = "/application/applicationsByUser/{username}"

var ParamId = "id"
var ParamName = "name"
var ParamTenantID = "tenantID"
var ParamUsername = "username"

const ResultProperty = "applications"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
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

	// Pagination options
	pagination.PaginationOptions
}

// List all applications on your tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
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
func (s *Service) ListByName(ctx context.Context, opt ListByNameOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.ListByNameB(opt))
}

func (s *Service) ListByNameB(opt ListByNameOptions) *core.TryRequest {
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
func (s *Service) ListByTenant(ctx context.Context, opt ListByTenantOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.ListByTenantB(opt))
}

func (s *Service) ListByTenantB(opt ListByTenantOptions) *core.TryRequest {
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
func (s *Service) ListByOwner(ctx context.Context, opt ListByOwnerOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.ListByOwnerB(opt))
}

func (s *Service) ListByOwnerB(opt ListByOwnerOptions) *core.TryRequest {
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
func (s *Service) ListByUser(ctx context.Context, opt ListByUserOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.ListByUserB(opt))
}

func (s *Service) ListByUserB(opt ListByUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamUsername, opt.Username).
		SetURL(ApiApplicationByUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an application
func (s *Service) Get(ctx context.Context, ID string) (*model.Application, error) {
	return core.ExecuteResultOnly[model.Application](ctx, s.GetB(ID))
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// Create an application
func (s *Service) Create(ctx context.Context, body any) (*model.Application, error) {
	return core.ExecuteResultOnly[model.Application](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplications)
	return core.NewTryRequest(s.Client, req)
}

// Update an application
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.Binary, error) {
	return core.ExecuteResultOnly[model.Binary](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
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

// Delete an application
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(ID, opt))
}

func (s *Service) DeleteB(ID string, opt DeleteOptions) *core.TryRequest {
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

// Copy an application (by a given ID)
func (s *Service) Copy(ctx context.Context, ID string, opt CopyOptions) (*model.ApplicationCollection, error) {
	return core.ExecuteResultOnly[model.ApplicationCollection](ctx, s.CopyB(ID, opt))
}

func (s *Service) CopyB(ID string, opt CopyOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, ID).
		SetURL(ApiApplicationClone)
	return core.NewTryRequest(s.Client, req)
}
