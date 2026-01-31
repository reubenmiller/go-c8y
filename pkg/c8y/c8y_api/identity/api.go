package identity

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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

func (s *Service) ListB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, id).
		SetURL(ApiIdentities)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

func (s *Service) GetB(opts IdentityOptions) *core.TryRequest {
	opts.withDefaults()
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamExternalID, opts.ExternalID).
		SetPathParam(ParamType, opts.Type).
		SetURL(ApiIdentity)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) CreateB(id string, opts IdentityOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, id).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(opts.withDefaults()).
		SetURL(ApiIdentities)
	return core.NewTryRequest(s.Client, req)
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
