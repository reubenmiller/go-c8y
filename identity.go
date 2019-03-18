package c8y

import (
	"context"
	"fmt"

	"github.com/tidwall/gjson"
)

// IdentityService does something
type IdentityService service

// IdentityOptions Identity parameters required when creating a new externalid
type IdentityOptions struct {
	ExternalID string `json:"externalId"`
	Type       string `json:"type"`
}

// Identity Cumulocity Identity object holding the information about the external id and link to the managed object
type Identity struct {
	ExternalID    string            `json:"externalId"`
	Type          string            `json:"type"`
	Self          string            `json:"self"`
	ManagedObject IdentityReference `json:"managedObject"`

	Item gjson.Result `json:"-"`
}

// IdentityReference contains the id and self link to the identify resource
type IdentityReference struct {
	ID   string `json:"id"`
	Self string `json:"self"`
}

// GetExternalID Get a managed object by an external ID
func (s *IdentityService) GetExternalID(ctx context.Context, identityType string, externalID string) (*Identity, *Response, error) {
	data := new(Identity)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("identity/externalIds/%s/%s", identityType, externalID),
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}

// Create adds a new external id for the given managed object id
func (s *IdentityService) Create(ctx context.Context, ID string, identity *IdentityOptions) (*Identity, *Response, error) {
	data := new(Identity)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         fmt.Sprintf("identity/globalIds/%s/externalIds", ID),
		Body:         identity,
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}
