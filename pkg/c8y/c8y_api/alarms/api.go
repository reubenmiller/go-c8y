package alarms

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiAlarms = "/alarm/alarms"
var ApiAlarmsCount = "/alarm/alarms/count"
var ApiAlarm = "/alarm/alarms/{id}"

var ParamId = "id"

const ResultProperty = "alarms"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service core.Service

// ListOptions to use when search for alarms
type ListOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the last update made
	LastUpdatedFrom time.Time `url:"lastUpdatedFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	LastUpdatedTo time.Time `url:"lastUpdatedTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`

	pagination.PaginationOptions
}

// List alarms
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.AlarmCollection, error) {
	return core.ExecuteResultOnly[model.AlarmCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// CountOptions to use when counting the active alarms
type CountOptions struct {
	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`

	pagination.PaginationOptions
}

// Get an alarm
func (s *Service) Get(ctx context.Context, ID string) (*model.Alarm, error) {
	return core.ExecuteResultOnly[model.Alarm](ctx, s.GetB(ID))
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiAlarm)
	return core.NewTryRequest(s.Client, req)
}

// Create an alarm
func (s *Service) Create(ctx context.Context, body any) (*model.Alarm, error) {
	return core.ExecuteResultOnly[model.Alarm](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}

// Update an alarm
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.Alarm, error) {
	return core.ExecuteResultOnly[model.Alarm](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarm)
	return core.NewTryRequest(s.Client, req)
}

// UpdateOptions option when bulk updating alarms
type BulkUpdateOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// TODO: Check if this is supported or not
	// The types of alarm to search for
	// Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`
}

// BulkUpdateAlarms bulk update of alarm collection
// The PUT method allows for updating alarms collections. Currently only the status of alarms can be changed.
// Response status:
// 200 - if the process has completed, all alarms have been updated
// 202 - if process continues in background
//
// Since this operations can take a lot of time, request returns after maximum 0.5 sec of processing, and updating is continued as a background process in the platform.
func (s *Service) UpdateList(ctx context.Context, opt BulkUpdateOptions, body any) (*model.AlarmCollection, error) {
	return core.ExecuteResultOnly[model.AlarmCollection](ctx, s.UpdateListB(opt, body))
}

func (s *Service) UpdateListB(opt BulkUpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(opt).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// DeleteListOptions option when deleting a collection of alarms
type DeleteListOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// TODO: Check if this is supported or not
	// The types of alarm to search for
	// Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`
}

// Remove alarm collections specified by query parameters
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteListB(opt))
}

func (s *Service) DeleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}
