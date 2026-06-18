// Package loggers manages the runtime log levels of a microservice via the
// Spring Boot actuator endpoints that Cumulocity proxies at
// /service/{name}/loggers. This only works for Spring Boot microservices based
// on the Cumulocity Java Microservice SDK.
package loggers

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	// ApiLoggers lists the loggers of a microservice.
	ApiLoggers = "/service/{name}/loggers"
	// ApiLogger addresses a single logger (by qualified package/class name).
	ApiLogger = "/service/{name}/loggers/{loggerName}"
)

const (
	ParamName       = "name"
	ParamLoggerName = "loggerName"

	// ResultProperty is the field in the /loggers response that holds the
	// logger-name -> configuration map.
	ResultProperty = "loggers"
)

// Service manages microservice log levels.
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
}

// List returns the configured loggers of a microservice. The server answers
// with {levels, loggers, groups}; the "loggers" map is plucked and rendered
// as-is (it is keyed by logger name, not an array).
func (s *Service) List(ctx context.Context, name string) op.Result[jsonmodels.MicroserviceLogger] {
	return core.ExecuteCollection(ctx, s.listB(name), ResultProperty, "", jsonmodels.NewMicroserviceLogger)
}

func (s *Service) listB(name string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, name).
		SetURL(ApiLoggers)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get returns the configured/effective log level of a single logger (a
// qualified package or class name).
func (s *Service) Get(ctx context.Context, name string, loggerName string) op.Result[jsonmodels.MicroserviceLogger] {
	return core.Execute(ctx, s.getB(name, loggerName), jsonmodels.NewMicroserviceLogger)
}

func (s *Service) getB(name string, loggerName string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, name).
		SetPathParam(ParamLoggerName, loggerName).
		SetURL(ApiLogger)
	return core.NewTryRequest(s.Client, req)
}

// Set configures the log level of a logger by POSTing the given body (e.g.
// {"configuredLevel":"DEBUG"}). Passing {"configuredLevel":null} resets the
// logger to its default level (the CLI delete command). The endpoint returns no
// body on success.
func (s *Service) Set(ctx context.Context, name string, loggerName string, body any) op.Result[jsonmodels.MicroserviceLogger] {
	return core.Execute(ctx, s.setB(name, loggerName, body), jsonmodels.NewMicroserviceLogger)
}

func (s *Service) setB(name string, loggerName string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, name).
		SetPathParam(ParamLoggerName, loggerName).
		SetURL(ApiLogger)
	return core.NewTryRequest(s.Client, req)
}
