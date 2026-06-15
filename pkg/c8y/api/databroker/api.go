// Package databroker provides access to Cumulocity data-broker connectors.
package databroker

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiConnectors = "/databroker/connectors"
var ApiConnector = "/databroker/connectors/{id}"

const ParamID = "id"

const ResultProperty = "connectors"

// Service interacts with data-broker connectors.
type Service struct {
	core.Service
}

// NewService creates a data-broker service.
func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// ConnectorIterator iterates data-broker connectors.
type ConnectorIterator = pagination.Iterator[jsonmodels.DatabrokerConnector]

// ListOptions filters data-broker connectors.
type ListOptions struct {
	pagination.PaginationOptions
}

// List returns a page of data-broker connectors.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DatabrokerConnector] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewDatabrokerConnector)
}

// ListAll returns an iterator over all data-broker connectors.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ConnectorIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.DatabrokerConnector] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewDatabrokerConnector,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConnectors)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get returns a data-broker connector by id.
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.DatabrokerConnector] {
	return core.Execute(ctx, s.getB(id), jsonmodels.NewDatabrokerConnector)
}

func (s *Service) getB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamID, id).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConnector)
	return core.NewTryRequest(s.Client, req)
}

// Update updates a data-broker connector by id (e.g. its status).
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.DatabrokerConnector] {
	return core.Execute(ctx, s.updateB(id, body), jsonmodels.NewDatabrokerConnector)
}

func (s *Service) updateB(id string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamID, id).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConnector)
	return core.NewTryRequest(s.Client, req)
}
