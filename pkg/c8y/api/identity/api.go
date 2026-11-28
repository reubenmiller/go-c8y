package identity

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"resty.dev/v3"
)

var apiIdentities = "/identity/globalIds/{id}/externalIds"
var apiIdentity = "/identity/externalIds/{type}/{externalID}"
var apiSearch = "/identity/search"

var paramID = "id"
var paramType = "type"
var paramExternalID = "externalID"

var DefaultType = "c8y_Global"

const resultProperty = "externalIds"

// Service provides api to get/set/delete external identities in Cumulocity
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
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

func (s *Service) listB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetURL(apiIdentities)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

func (s *Service) getB(opts IdentityOptions) *httpcore.TryRequest {
	opts.withDefaults()
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramExternalID, opts.ExternalID).
		SetPathParam(paramType, opts.Type).
		SetURL(apiIdentity)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) createB(id string, opts IdentityOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(paramID, id).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(opts.withDefaults()).
		SetURL(apiIdentities)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// SearchOptions is the request body for a bulk external-ID search (POST /identity/search).
type SearchOptions struct {
	// ExternalIds is the list of external identifiers to look up.
	ExternalIds []IdentityOptions `json:"externalIds"`
}

func (s *Service) searchB(opts SearchOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(opts).
		SetURL(apiSearch)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

func (s *Service) deleteB(opts IdentityOptions) *httpcore.TryRequest {
	opts.withDefaults()
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramType, opts.Type).
		SetPathParam(paramExternalID, opts.ExternalID).
		SetURL(apiIdentity)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
