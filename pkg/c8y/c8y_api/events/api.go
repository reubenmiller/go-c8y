package events

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/events/eventbinaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiEvents = "/event/events"
var ApiEvent = "/event/events/{id}"

var ParamId = "id"

const ResultProperty = "events"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:  *s,
		Binaries: eventbinaries.NewService(s),
	}
}

// Service provides api to get/set/delete events in Cumulocity
type Service struct {
	core.Service
	Binaries *eventbinaries.Service
}

// ListOptions to use when search for events
type ListOptions struct {
	// Start date or date and time of the event's creation (set by the platform during creation)
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the event's creation (set by the platform during creation)
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the last update made
	LastUpdatedFrom time.Time `url:"lastUpdatedFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	LastUpdatedTo time.Time `url:"lastUpdatedTo,omitempty,omitzero"`

	// Start date or date and time of the event occurrence (provided by the device).
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	FragmentType string `url:"fragmentType,omitempty"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	// Important: Only fragments with a string value are supported.
	FragmentValue string `url:"fragmentValue,omitempty"`

	// If you are using a range query (that is, at least one of
	// the dateFrom or dateTo parameters is included in the request),
	// then setting revert=true will sort the results by the oldest
	// events first. By default, the results are sorted by the newest
	// events first.
	Revert bool `url:"revert,omitempty"`

	// The managed object ID to which the event is associated
	Source string `url:"source,omitempty"`

	// SourceRef allows resolving the source from various references (external ID, name, query, etc.)
	// If set, this takes precedence over Source field
	SourceRef source.Resolver `url:"-"`

	// The type of event to search for
	Type string `url:"type,omitempty"`

	// When set to true, events for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, events for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, events for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, events for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`

	pagination.PaginationOptions
}

// Resolve resolves all reference fields (SourceRef) to their concrete values.
// Only resolves if the direct field (Source) is not already set.
func (opt *ListOptions) Resolve(ctx context.Context) error {
	if opt.SourceRef != nil && opt.Source == "" {
		result, err := opt.SourceRef.ResolveID(ctx)
		if err != nil {
			return err
		}
		opt.Source = result.ID
	}
	return nil
}

// EventIterator provides iteration over events
type EventIterator = pagination.Iterator[jsonmodels.Event]

// List events
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Event] {
	if err := opt.Resolve(ctx); err != nil {
		return op.Failed[jsonmodels.Event](err, true)
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewEvent)
}

// ListAll returns an iterator for all events
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *EventIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.Event] {
		return s.List(ctx, opts)
	}, jsonmodels.NewEvent)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an event
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewEvent)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiEvent)
	return core.NewTryRequest(s.Client, req)
}

// Create an event
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewEvent)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req)
}

// Update an event
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewEvent)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiEvent)
	return core.NewTryRequest(s.Client, req)
}

// DeleteListOptions option when deleting a collection of events
type DeleteListOptions struct {
	// Start date or date and time of the event's creation (set by the platform during creation)
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the event's creation (set by the platform during creation)
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the event occurrence (provided by the device).
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	FragmentType string `url:"fragmentType,omitempty"`

	// The managed object ID to which the event is associated
	Source string `url:"source,omitempty"`

	// The type of event to search for
	Type string `url:"type,omitempty"`
}

// Remove event collections specified by query parameters
//
// DELETE requests are not synchronous. The response could be returned
// before the delete request has been completed. This may happen especially
// when the deleted event has a lot of associated data. After sending the
// request, the platform starts deleting the associated data in an asynchronous way.
// Finally, the requested event is deleted after all associated data has been deleted.
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.deleteListB(opt), jsonmodels.NewEvent)
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req)
}
