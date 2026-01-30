package softwareitems

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamId = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:   *s,
		inventory: managedobjects.NewService(s),
	}
}

// Service api to interact with software items
// type Service core.Service
type Service struct {
	core.Service
	inventory *managedobjects.Service
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
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.SoftwareCollection, error) {
	return core.ExecuteResultOnly[model.SoftwareCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	// inventoryQuery := model.InventoryQuery{}
	return s.inventory.ListB(managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddOrderBy("name").
			AddOrderBy("creationTime").
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", opt.Name).
			AddFilterEqStr("softwareType", opt.SoftwareType).
			AddFilterEqStr("c8y_Filter.type", opt.DeviceType).
			Build(),
		PaginationOptions: opt.PaginationOptions,
	})
}

type GetOptions struct {
	ID                string `url:"-"`
	WithParents       bool   `url:"withParents,omitempty"`
	WithChildren      bool   `url:"withChildren,omitempty"`
	withChildrenCount bool   `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool   `url:"skipChildrenNames,omitempty"`
}

// Create a software item
func (s *Service) Create(ctx context.Context, body any) (*model.Software, error) {
	return core.ExecuteResultOnly[model.Software](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req)
}

type GetOrCreateOptions struct {
	Software model.Software
}

func (s *Service) GetOrCreate(ctx context.Context, opt GetOrCreateOptions) (*model.Software, bool, error) {
	return pagination.FindOrCreate[model.Software](
		ctx,
		s.ListB(ListOptions{
			Name:         opt.Software.Name,
			SoftwareType: opt.Software.SoftwareType,
		}),
		s.CreateB(opt.Software),
		pagination.DefaultSearch(),
	)
}

// Get a software item
func (s *Service) Get(ctx context.Context, opt GetOptions) (*model.Software, error) {
	return core.ExecuteResultOnly[model.Software](ctx, s.GetB(opt))
}

func (s *Service) GetB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, opt.ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// Update a software item
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.Software, error) {
	return core.ExecuteResultOnly[model.Software](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a software item
type DeleteOptions struct {
	ID string `url:"-"`

	// When set to true all the hierarchy will be deleted without checking the type of managed object. It takes precedence over the parameter cascade
	// ForceCascade *bool `url:"forceCascade,omitempty"`

	SkipCascade bool `url:"-"`
}

// Delete a software item
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(opt))
}

func (s *Service) DeleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, opt.ID).
		SetQueryParam("forceCascade", fmt.Sprintf("%v", !opt.SkipCascade)).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}
