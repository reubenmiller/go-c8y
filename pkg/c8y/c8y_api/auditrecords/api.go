package auditrecords

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"resty.dev/v3"
)

var ApiAuditRecords = "/audit/auditRecords"
var ApiAuditRecord = "/audit/auditRecords/{id}"

var ParamId = "id"

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

	Revert bool `url:"revert,omitempty"`

	pagination.PaginationOptions
}

// List the audit records
func (s *Service) List(ctx context.Context, opt ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAuditRecords)
}

// Get an audit record
func (s *Service) Get(ctx context.Context, ID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiAuditRecord)
}

// Create an audit record
func (s *Service) Create(ctx context.Context, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiAuditRecords)
}
