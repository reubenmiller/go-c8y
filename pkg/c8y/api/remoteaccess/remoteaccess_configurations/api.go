package remoteaccess_configurations

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiConfigurations = "/service/remoteaccess/devices/{managedObjectID}/configurations"
var ApiConfiguration = "/service/remoteaccess/devices/{managedObjectID}/configurations/{id}"

var ParamId = "id"
var ParamManagedObjectID = "managedObjectID"

// Service provides api to managed Cloud Remote Access configurations
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
}

type ListOptions struct {
	ManagedObjectID  string `url:"-"`
	ManagedObjectRef source.Resolver
}

// Resolve resolves all reference fields (ApplicationRef) to their concrete values.
// Only resolves if the direct field (ID) is not already set.
func (opt *ListOptions) Resolve(ctx context.Context) error {
	if opt.ManagedObjectRef != nil && opt.ManagedObjectID == "" {
		result, err := opt.ManagedObjectRef.ResolveID(ctx)
		if err != nil {
			return err
		}
		opt.ManagedObjectID = result.ID
	}
	return nil
}

// List remote access configurations
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	if err := opt.Resolve(ctx); err != nil {
		return op.Failed[jsonmodels.RemoteAccessConfiguration](err, true)
	}
	return core.ExecuteCollection(ctx, s.listB(opt), "", "", jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfigurations)
	return core.NewTryRequest(s.Client, req)
}

type GetOptions struct {
	ManagedObjectID string `url:"-"`
	ConfigurationID string `url:"-"`
}

// Get remote access configuration
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamId, opt.ConfigurationID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfiguration)
	return core.NewTryRequest(s.Client, req)
}

type CreateOptions struct {
	ManagedObjectID string
	Body            any
}

// Create remote access configuration
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return core.Execute(ctx, s.createB(opt, opt.Body), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) createB(opt CreateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiConfigurations)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOptions struct {
	ManagedObjectID string
	ConfigurationID string
	Body            any
}

// Update remote access configuration
func (s *Service) Update(ctx context.Context, opt UpdateOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return core.Execute(ctx, s.updateB(opt), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) updateB(opt UpdateOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamId, opt.ConfigurationID).
		SetBody(opt.Body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfiguration)
	return core.NewTryRequest(s.Client, req)
}

type DeleteOptions struct {
	ManagedObjectID string
	ConfigurationID string
}

// Delete remote access configuration
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(opt))
}

func (s *Service) deleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamId, opt.ConfigurationID).
		SetURL(ApiConfiguration)
	return core.NewTryRequest(s.Client, req)
}
