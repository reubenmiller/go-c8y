package auditrecords

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiAuditRecords = "/audit/auditRecords"
var apiAuditRecord = "/audit/auditRecords/{id}"

var paramID = "id"

const resultProperty = "auditRecords"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
}

// ListOptions is generated from the OpenAPI spec — see zz_generated_options.go.
// (Revert is added via the overlay's extraFields: the server supports it but the OAS
// omits the parameter.)

// AuditRecordIterator provides iteration over audit records
type AuditRecordIterator = pagination.Iterator[jsonmodels.AuditRecord]

// List the audit records
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.AuditRecord] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewAuditRecord)
}

// ListAll returns an iterator for all audit records
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *AuditRecordIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.AuditRecord] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewAuditRecord,
	)
}

func (s *Service) listB(opt any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiAuditRecords)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Get an audit record
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.AuditRecord] {
	return httpcore.Execute(ctx, s.getB(ID), jsonmodels.NewAuditRecord)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramID, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiAuditRecord)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create an audit record
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.AuditRecord] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewAuditRecord)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiAuditRecords)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
