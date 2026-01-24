package identity

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"resty.dev/v3"
)

var ApiIdentities = "/identity/globalIds/{id}/externalIds"
var ApiIdentity = "/identity/externalIds/{type}/{externalID}"

var ParamId = "id"
var ParamType = "type"
var ParamExternalID = "externalID"

var DefaultType = "c8y_Global"

const ResultProperty = "externalIds"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

// IdentityOptions identity options
type IdentityOptions struct {
	// External Identity
	ExternalID string `json:"externalId,omitempty"`

	// External ID Type. Defaults to c8y_Serial if not set
	Type string `json:"type,omitempty"`
}

func (i *IdentityOptions) withDefaults() *IdentityOptions {
	if i.Type == "" {
		i.Type = DefaultType
	}
	return i
}

func (i *IdentityOptions) GetType() string {
	if i.Type == "" {
		return DefaultType
	}
	return string(i.Type)
}

// List the external identities of a managed object
func (s *Service) List(ctx context.Context, id string) (*model.IdentityCollection, error) {
	return core.ExecuteResultOnly[model.IdentityCollection](ctx, s.ListB(id))
}

func (s *Service) ListB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, id).
		SetURL(ApiIdentities)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an identity
func (s *Service) Get(ctx context.Context, opts IdentityOptions) (*model.Identity, error) {
	return core.ExecuteResultOnly[model.Identity](ctx, s.GetB(opts))
}

func (s *Service) GetB(opts IdentityOptions) *core.TryRequest {
	opts.withDefaults()
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamExternalID, opts.ExternalID).
		SetPathParam(ParamType, opts.Type).
		SetURL(ApiIdentity)
	return core.NewTryRequest(s.Client, req)
}

// Create an identity
func (s *Service) Create(ctx context.Context, id string, opts IdentityOptions) (*model.Identity, error) {
	return core.ExecuteResultOnly[model.Identity](ctx, s.CreateB(id, opts))
}

func (s *Service) CreateB(id string, opts IdentityOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, id).
		SetBody(opts.withDefaults()).
		SetURL(ApiIdentities)
	return core.NewTryRequest(s.Client, req)
}

// Delete an identity
func (s *Service) Delete(ctx context.Context, opts IdentityOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(opts))
}

func (s *Service) DeleteB(opts IdentityOptions) *core.TryRequest {
	opts.withDefaults()
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamType, opts.Type).
		SetPathParam(ParamExternalID, opts.ExternalID).
		SetURL(ApiIdentity)
	return core.NewTryRequest(s.Client, req)
}
