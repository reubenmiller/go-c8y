package mock

import (
	"embed"
	"strings"
)

//go:embed responses/*.json
var responses embed.FS

// ResponseType represents the type of mock response
type ResponseType string

const (
	ManagedObject               ResponseType = "managedobject"
	ManagedObjectCollection     ResponseType = "managedobject_collection"
	Alarm                       ResponseType = "alarm"
	AlarmCollection             ResponseType = "alarm_collection"
	Event                       ResponseType = "event"
	EventCollection             ResponseType = "event_collection"
	Operation                   ResponseType = "operation"
	OperationCollection         ResponseType = "operation_collection"
	Measurement                 ResponseType = "measurement"
	MeasurementCollection       ResponseType = "measurement_collection"
	MeasurementSeriesCollection ResponseType = "measurement_series_collection"
	Application                 ResponseType = "application"
	ApplicationCollection       ResponseType = "application_collection"
)

// GetResponse returns the mock response for the given type
func GetResponse(responseType ResponseType) ([]byte, error) {
	return responses.ReadFile("responses/" + string(responseType) + ".json")
}

// DetectResponseType attempts to detect the response type from the request URL path
func DetectResponseType(urlPath string, isCollection bool) ResponseType {
	path := strings.ToLower(urlPath)

	// Check for specific endpoints
	switch {
	case strings.Contains(path, "/inventory/managedobjects"):
		if isCollection {
			return ManagedObjectCollection
		}
		return ManagedObject
	case strings.Contains(path, "/alarm/alarms"):
		if isCollection {
			return AlarmCollection
		}
		return Alarm
	case strings.Contains(path, "/event/events"):
		if isCollection {
			return EventCollection
		}
		return Event
	case strings.Contains(path, "/devicecontrol/operations"):
		if isCollection {
			return OperationCollection
		}
		return Operation
	case strings.Contains(path, "/measurement/measurements/series"):
		if isCollection {
			return MeasurementSeriesCollection
		}
		return Measurement
	case strings.Contains(path, "/measurement/measurements"):
		if isCollection {
			return MeasurementCollection
		}
		return Measurement
	case strings.Contains(path, "/application/applications"):
		if isCollection {
			return ApplicationCollection
		}
		return Application
	}

	// Default to managed object
	if isCollection {
		return ManagedObjectCollection
	}
	return ManagedObject
}
