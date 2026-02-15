package model

import (
	"time"
)

// Event entity
type Event struct {
	ID     string    `json:"id,omitempty"`
	Source *Source   `json:"source,omitempty"`
	Type   string    `json:"type,omitempty"`
	Text   string    `json:"text,omitempty"`
	Self   string    `json:"self,omitempty"`
	Time   time.Time `json:"time,omitempty,omitzero"`
}

// EventCollection collection of events
type EventCollection struct {
	*BaseResponse

	Events []Event `json:"events"`
}

// EventBinary binary object associated with an event
type EventBinary struct {
	// A URL linking to this resource
	Self string `json:"self,omitempty"`

	// Media type of the attachment
	Type string `json:"type,omitempty"`

	// Unique identifier of the event
	Source string `json:"source,omitempty"`

	// Name of the attachment. If it is not provided in the request, it will be set as the event ID.
	Name string `json:"name,omitempty"`
}
