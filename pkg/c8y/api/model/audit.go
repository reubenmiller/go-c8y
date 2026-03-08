package model

import (
	"time"
)

// AuditSeverity represents the severity level of an audit record
type AuditSeverity string

const (
	AuditSeverityCritical AuditSeverity = "CRITICAL"
	AuditSeverityMajor    AuditSeverity = "MAJOR"
	AuditSeverityMinor    AuditSeverity = "MINOR"
	AuditSeverityWarning  AuditSeverity = "WARNING"
)

// AuditRecord representation
type AuditRecord struct {
	ID           string        `json:"id,omitempty"`
	Self         string        `json:"self,omitempty"`
	CreationTime time.Time     `json:"creationTime,omitempty,omitzero"`
	Type         string        `json:"type,omitempty"`
	Time         time.Time     `json:"time,omitempty,omitzero"`
	Text         string        `json:"text,omitempty"`
	Source       *Source       `json:"source,omitempty"`
	User         string        `json:"user,omitempty"`
	Application  string        `json:"application,omitempty"`
	Activity     string        `json:"activity,omitempty"`
	Severity     AuditSeverity `json:"severity,omitempty"`
	// Changes     []ChangeDescription     `json:"changes,omitempty"`
}

// AuditRecordsCollection collection of audit records
type AuditRecordsCollection struct {
	*BaseResponse

	AuditRecords []AuditRecord `json:"auditRecords,omitempty"`
}
