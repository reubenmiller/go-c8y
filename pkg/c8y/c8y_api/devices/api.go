package devices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/devices/enrollment"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/devices/registration"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiDeviceControlAccessToken = "/devicecontrol/deviceAccessToken"

const ParamId = "id"

const ResultProperty = managedobjects.ResultProperty

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		Enrollment:     enrollment.NewService(s),
		Registration:   registration.NewService(s),
		managedObjects: *managedobjects.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service

	Enrollment   *enrollment.Service
	Registration *registration.Service

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

// ManagedObjectIterator provides iteration over managed objects
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all devices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.ManagedObject] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewManagedObject,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
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
	return core.ExecuteCollection(ctx, s.findB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

func (s *Service) findB(opt FindOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}
