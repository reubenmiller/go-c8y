package c8y

import (
	"context"
	"fmt"
	"log"
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
	ID     string `json:"id"`
	Source struct {
		Self string `json:"self"`
		ID   string `json:"id"`
	} `json:"source"`
	Type string    `json:"type"`
	Text string    `json:"text"`
	Self string    `json:"self"`
	Time Timestamp `json:"time"`
}

// EventCollection todo
type EventCollection struct {
	*C8yBaseResponse

	Events []EventObject `json:"events"`
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

	if data.C8yBaseResponse.Statistics.TotalPages != nil {
		log.Printf("Total events: %d\n", *data.C8yBaseResponse.Statistics.TotalPages)
	}

	return data, resp, nil
}
