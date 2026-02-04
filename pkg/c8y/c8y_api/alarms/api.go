package alarms

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
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
type Service struct {
	core.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new alarms service with device resolution capabilities
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

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

	// Source device to filter measurements by.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
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

// AlarmIterator provides iteration over alarms
type AlarmIterator = pagination.Iterator[jsonmodels.Alarm]

// List alarms
// The Source field supports resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Alarm] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewAlarm)
}

// ListAll returns an iterator for all alarms
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *AlarmIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Alarm] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewAlarm,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
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
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Alarm] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewAlarm)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiAlarm)
	return core.NewTryRequest(s.Client, req)
}

// Create an alarm
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Alarm] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewAlarm)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}

// Update an alarm
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Alarm] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewAlarm)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
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
func (s *Service) UpdateList(ctx context.Context, opt BulkUpdateOptions, body any) op.Result[jsonmodels.Alarm] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
		opt.Source = resolvedID
	}

	return core.Execute(ctx, s.updateListB(opt, body), jsonmodels.NewAlarm)
}

func (s *Service) updateListB(opt BulkUpdateOptions, body any) *core.TryRequest {
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
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[core.NoContent] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[core.NoContent](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteNoContent(ctx, s.deleteListB(opt))
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}
