package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// MicroserviceLogger represents the log-level configuration of a microservice
// logger, as returned by the Spring Boot actuator endpoints proxied through
// Cumulocity at /service/{name}/loggers[/{loggerName}].
//
// A single logger response looks like {"configuredLevel":"DEBUG",
// "effectiveLevel":"DEBUG"}; the collection response plucks the "loggers" map
// (logger name -> configuration), which is rendered as-is.
type MicroserviceLogger struct {
	jsondoc.Facade
}

func NewMicroserviceLogger(b []byte) MicroserviceLogger {
	return MicroserviceLogger{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ConfiguredLevel is the explicitly configured log level (empty when only the
// effective level is set).
func (m MicroserviceLogger) ConfiguredLevel() string {
	return m.Get("configuredLevel").String()
}

// EffectiveLevel is the log level actually in effect for the logger.
func (m MicroserviceLogger) EffectiveLevel() string {
	return m.Get("effectiveLevel").String()
}
