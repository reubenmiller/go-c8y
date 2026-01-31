package devices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"resty.dev/v3"
)

var ApiDeviceControlAccessToken = "/devicecontrol/deviceAccessToken"

const ParamId = "id"

const ResultProperty = managedobjects.ResultProperty

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: *managedobjects.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service

	managedObjects managedobjects.Service
}

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), managedobjects.ResultProperty, "", jsonmodels.NewManagedObject)
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// FindOptions filter devices
type FindOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) Find(ctx context.Context, opt FindOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.FindB(opt), managedobjects.ResultProperty, managedobjects.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

func (s *Service) FindB(opt FindOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}
