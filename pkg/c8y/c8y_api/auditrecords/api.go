package auditrecords

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiAuditRecords = "/audit/auditRecords"
var ApiAuditRecord = "/audit/auditRecords/{id}"

var ParamId = "id"

const ResultProperty = "auditRecords"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service core.Service

// ListOptions to use when search for audit entries
type ListOptions struct {
	// Start date or date and time of the audit record (device time).
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the audit record (device time).
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The type of audit record to search for.
	Type string `url:"type,omitempty"`

	// The platform component ID to which the audit is associated.
	Source string `url:"source,omitempty"`

	// Name of the application from which the audit was carried out
	Application string `url:"application,omitempty"`

	// The username to search for.
	User string `url:"user,omitempty"`

	// TODO: Check if this is supported or not
	// https://cumulocity.com/api/core/#operation/getAuditRecordCollectionResource
	Revert bool `url:"revert,omitempty"`

	pagination.PaginationOptions
}

// AuditRecordIterator provides iteration over audit records
type AuditRecordIterator = pagination.Iterator[jsonmodels.AuditRecord]

// List the audit records
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.AuditRecord] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewAuditRecord)
}

// ListAll returns an iterator for all audit records
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *AuditRecordIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return pagination.Paginate(ctx, func(page int) op.Result[jsonmodels.AuditRecord] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, jsonmodels.NewAuditRecord, opts.GetMaxItems())
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAuditRecords)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an audit record
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.AuditRecord] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID), jsonmodels.NewAuditRecord)
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiAuditRecord)
	return core.NewTryRequest(s.Client, req)
}

// Create an audit record
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.AuditRecord] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewAuditRecord)
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiAuditRecords)
	return core.NewTryRequest(s.Client, req)
}
