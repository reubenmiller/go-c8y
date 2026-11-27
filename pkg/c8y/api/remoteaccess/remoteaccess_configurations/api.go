package remoteaccess_configurations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
	"resty.dev/v3"
)

var ApiConfigurations = "/service/remoteaccess/devices/{managedObjectID}/configurations"
var ApiConfiguration = "/service/remoteaccess/devices/{managedObjectID}/configurations/{id}"

var ParamID = "id"
var ParamManagedObjectID = "managedObjectID"

// Service provides api to managed Cloud Remote Access configurations
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
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
	return httpcore.ExecuteCollection(ctx, s.listB(opt), "", "", jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfigurations)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type GetOptions struct {
	ManagedObjectID string `url:"-"`
	ConfigurationID string `url:"-"`
}

// Get remote access configuration
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamID, opt.ConfigurationID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfiguration)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ResolveID resolves a configuration reference to its id, scoped to a device.
// A plain id (or an empty reference) passes through unchanged with no lookup; a
// "name:<name>" reference is matched case-insensitively against the device's
// configurations (first match wins), mirroring the v1 by-name resolver. The
// managedObjectID must already be a concrete id.
func (s *Service) ResolveID(ctx context.Context, managedObjectID, ref string) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("remote access configuration", ref, resolveErr) }()
	name, isName := strings.CutPrefix(ref, "name:")
	if !isName {
		return ref, nil
	}
	res := s.List(ctx, ListOptions{ManagedObjectID: managedObjectID})
	if res.Err != nil {
		return "", res.Err
	}
	for cfg, err := range res.Items() {
		if err != nil {
			return "", err
		}
		if strings.EqualFold(cfg.Name(), name) {
			return cfg.ID(), nil
		}
	}
	return "", &ConfigurationNotFoundError{Name: name}
}

// ErrConfigurationNotFound is matched (errors.Is) by the error [Service.ResolveID]
// returns when no configuration of the device has the requested name. The
// error is a [*ConfigurationNotFoundError], and also matches
// [core.ErrNameNotFound].
var ErrConfigurationNotFound = errors.New("remote access configuration not found")

// ConfigurationNotFoundError is the error of [Service.ResolveID] for a name
// which no configuration of the device has. Its text is
// "remote access configuration not found: name=<name>".
type ConfigurationNotFoundError struct {
	// Name is the name which matched nothing.
	Name string
}

func (e *ConfigurationNotFoundError) Error() string {
	return fmt.Sprintf("%s: name=%s", ErrConfigurationNotFound.Error(), e.Name)
}

// Is reports whether target is [ErrConfigurationNotFound] or [core.ErrNameNotFound].
func (e *ConfigurationNotFoundError) Is(target error) bool {
	return target == ErrConfigurationNotFound || target == core.ErrNameNotFound
}

// NotFoundName returns the resource and the name, see [core.NameNotFound].
func (e *ConfigurationNotFoundError) NotFoundName() (resource, name string) {
	return "remote access configuration", e.Name
}

var (
	// ErrPassthroughNotFound is matched (errors.Is) by the error [Service.Detect]
	// returns when no passthrough configuration has the requested id, name or
	// name pattern.
	ErrPassthroughNotFound = errors.New("remote access passthrough configuration not found")

	// ErrNoPassthrough is matched (errors.Is) by the error [Service.Detect]
	// returns when it was not given a name and the device has no passthrough
	// configuration.
	ErrNoPassthrough = errors.New("device has no remote access passthrough configuration")
)

// DetectOptions selects the passthrough configuration of a device for a
// remote access connection (see [Service.Detect]).
type DetectOptions struct {
	ManagedObjectID string
	// Name is a configuration id, a name (case-insensitive) or a name pattern
	// with * wildcards. Leave it empty to select a configuration automatically.
	Name string
}

// Detect returns the passthrough configuration of a device to connect to, as
// the local proxy and the ssh client need one.
//
// With a Name, it returns the configuration with that id, else the first
// passthrough configuration with that name, else the first whose name matches
// the Name as a wildcard pattern. Without a Name, it returns the only
// passthrough configuration, else the first on port 22, else the first with
// "ssh" in its name, else the first one. The managedObjectID must already be
// a concrete id.
func (s *Service) Detect(ctx context.Context, opt DetectOptions) (jsonmodels.RemoteAccessConfiguration, error) {
	if source.IsID(opt.Name) {
		res := s.Get(ctx, GetOptions{ManagedObjectID: opt.ManagedObjectID, ConfigurationID: opt.Name})
		if res.Err == nil {
			return res.Data, nil
		}
		// A name can look like an id, so match it by name below
	}

	res := s.List(ctx, ListOptions{ManagedObjectID: opt.ManagedObjectID})
	if res.Err != nil {
		return jsonmodels.RemoteAccessConfiguration{}, res.Err
	}
	passthrough := make([]jsonmodels.RemoteAccessConfiguration, 0)
	for cfg, err := range res.Items() {
		if err != nil {
			return jsonmodels.RemoteAccessConfiguration{}, err
		}
		if strings.EqualFold(cfg.Protocol(), string(model.RemoteAccessProtocolPassthrough)) {
			passthrough = append(passthrough, cfg)
		}
	}

	if opt.Name != "" {
		for _, cfg := range passthrough {
			if strings.EqualFold(cfg.Name(), opt.Name) {
				return cfg, nil
			}
		}
		for _, cfg := range passthrough {
			if ok, _ := matcher.MatchWithWildcards(cfg.Name(), opt.Name); ok {
				return cfg, nil
			}
		}
		return jsonmodels.RemoteAccessConfiguration{}, fmt.Errorf("%w: name=%s", ErrPassthroughNotFound, opt.Name)
	}

	switch len(passthrough) {
	case 0:
		return jsonmodels.RemoteAccessConfiguration{}, fmt.Errorf("%w: id=%s", ErrNoPassthrough, opt.ManagedObjectID)
	case 1:
		return passthrough[0], nil
	}
	for _, cfg := range passthrough {
		if cfg.Port() == 22 {
			return cfg, nil
		}
	}
	for _, cfg := range passthrough {
		if strings.Contains(strings.ToLower(cfg.Name()), "ssh") {
			return cfg, nil
		}
	}
	return passthrough[0], nil
}

type CreateOptions struct {
	ManagedObjectID string
	Body            any
}

// Create remote access configuration
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return httpcore.Execute(ctx, s.createB(opt, opt.Body), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) createB(opt CreateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiConfigurations)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateOptions struct {
	ManagedObjectID string
	ConfigurationID string
	Body            any
}

// Update remote access configuration
func (s *Service) Update(ctx context.Context, opt UpdateOptions) op.Result[jsonmodels.RemoteAccessConfiguration] {
	return httpcore.Execute(ctx, s.updateB(opt), jsonmodels.NewRemoteAccessConfiguration)
}

func (s *Service) updateB(opt UpdateOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamID, opt.ConfigurationID).
		SetBody(opt.Body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiConfiguration)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type DeleteOptions struct {
	ManagedObjectID string
	ConfigurationID string
}

// Delete remote access configuration
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(opt)).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamManagedObjectID, opt.ManagedObjectID).
		SetPathParam(ParamID, opt.ConfigurationID).
		SetURL(ApiConfiguration)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
