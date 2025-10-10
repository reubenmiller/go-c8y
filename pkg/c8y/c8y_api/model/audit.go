package model

import (
	"time"
)

// AuditRecord representation
type AuditRecord struct {
	ID           string    `json:"id,omitempty"`
	Self         string    `json:"self,omitempty"`
	CreationTime time.Time `json:"creationTime,omitempty,omitzero"`
	Type         string    `json:"type,omitempty"`
	Time         time.Time `json:"time,omitempty,omitzero"`
	Text         string    `json:"text,omitempty"`
	Source       *Source   `json:"source,omitempty"`
	User         string    `json:"user,omitempty"`
	Application  string    `json:"application,omitempty"`
	Activity     string    `json:"activity,omitempty"`
	Severity     string    `json:"severity,omitempty"`
	// Changes     []ChangeDescription     `json:"changes,omitempty"`
}
