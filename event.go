package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// EventService does something
type EventService service

// EventCollectionOptions todo
type EventCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	PaginationOptions
}

// Event todo
type Event struct {
	ID     string     `json:"id,omitempty"`
	Source *Source    `json:"source,omitempty"`
	Type   string     `json:"type,omitempty"`
	Text   string     `json:"text,omitempty"`
	Self   string     `json:"self,omitempty"`
	Time   *Timestamp `json:"time,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// EventCollection todo
type EventCollection struct {
	*BaseResponse

	Events []Event `json:"events"`

	// Allow access to custom fields
	Items []gjson.Result `json:"-"`
}

// GetEvent returns a new event object
func (s *EventService) GetEvent(ctx context.Context, ID string) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "event/events/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetEvents returns a list of events based on given filters
func (s *EventService) GetEvents(ctx context.Context, opt *EventCollectionOptions) (*EventCollection, *Response, error) {
	data := new(EventCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "event/events",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new event object
func (s *EventService) Create(ctx context.Context, body interface{}) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "event/events",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates properties on an existing event
func (s *EventService) Update(ctx context.Context, ID string, body interface{}) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "event/events/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete event by its ID
func (s *EventService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "event/events/" + ID,
	})
}

// DeleteEvents removes a collection of events based on the given filters
func (s *EventService) DeleteEvents(ctx context.Context, opt *EventCollectionOptions) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "event/events",
		Query:  opt,
	})
}
