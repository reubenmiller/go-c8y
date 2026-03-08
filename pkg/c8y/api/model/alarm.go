package model

import (
	"time"
)

// Cumulocity alarm Severity types
type AlarmSeverity string

const (
	AlarmSeverityCritical AlarmSeverity = "CRITICAL"
	AlarmSeverityMajor    AlarmSeverity = "MAJOR"
	AlarmSeverityMinor    AlarmSeverity = "MINOR"
	AlarmSeverityWarning  AlarmSeverity = "WARNING"
)

func NewAlarmSeverity(v ...AlarmSeverity) []AlarmSeverity {
	return v
}

// Cumulocity alarm status states
type AlarmStatus string

const (
	AlarmStatusActive       AlarmStatus = "ACTIVE"
	AlarmStatusAcknowledged AlarmStatus = "ACKNOWLEDGED"
	AlarmStatusCleared      AlarmStatus = "CLEARED"
)

// Alarm representation
type Alarm struct {
	ID                  string        `json:"id,omitempty"`
	Source              *Source       `json:"source,omitempty"`
	Type                string        `json:"type,omitempty"`
	Time                time.Time     `json:"time,omitempty,omitzero"`
	CreationTime        time.Time     `json:"creationTime,omitempty,omitzero"`
	FirstOccurrenceTime time.Time     `json:"firstOccurrenceTime,omitempty,omitzero"`
	Text                string        `json:"text,omitempty"`
	Status              AlarmStatus   `json:"status,omitempty"`
	Severity            AlarmSeverity `json:"severity,omitempty"`
	Count               uint64        `json:"count,omitempty"`
	Self                string        `json:"self,omitempty"`
}

// AlarmCollection collection of alarms
type AlarmCollection struct {
	*BaseResponse

	Alarms []Alarm `json:"alarms"`
}

// AlarmUpdateProperties properties which can be updated on an existing alarm
type AlarmUpdateProperties struct {
	Text     string        `json:"text,omitempty"`
	Status   AlarmStatus   `json:"status,omitempty"`
	Severity AlarmSeverity `json:"severity,omitempty"`
}
