package c8y

import (
	"context"
	"fmt"
	"log"

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

// EventObject todo
type EventObject struct {
	ID     string    `json:"id,omitempty"`
	Source Source    `json:"source,omitempty"`
	Type   string    `json:"type,omitempty"`
	Text   string    `json:"text,omitempty"`
	Self   string    `json:"self,omitempty"`
	Time   Timestamp `json:"time,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// EventCollection todo
type EventCollection struct {
	*BaseResponse

	Events []EventObject `json:"events"`

	// Allow access to custom fields
	Items []gjson.Result `json:"-"`
}

// GetEventCollection todo
func (s *EventService) GetEventCollection(ctx context.Context, opt *EventCollectionOptions) (*EventCollection, *Response, error) {
	u := fmt.Sprintf("event/events")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(EventCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	if data.BaseResponse.Statistics.TotalPages != nil {
		log.Printf("Total events: %d\n", *data.BaseResponse.Statistics.TotalPages)
	}

	data.Items = resp.JSON.Get("events").Array()
	return data, resp, nil
}

// CreateEvent creates a new event object
func (s *EventService) CreateEvent(ctx context.Context, body interface{}) (*EventObject, *Response, error) {
	data := new(EventObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "event/events",
		Body:         body,
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}
