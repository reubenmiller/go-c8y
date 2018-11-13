package c8y

import (
	"context"
	"fmt"
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
}

type IdentityReference struct {
	ID   string `json:"id"`
	Self string `json:"self"`
}

// GetExternalID Get a managed object by an external ID
func (s *IdentityService) GetExternalID(ctx context.Context, identityType string, externalID string) (*Identity, *Response, error) {
	u := fmt.Sprintf("identity/externalIds/%s/%s", identityType, externalID)

	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(Identity)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// NewExternalIdentity Creates a new external id for the given managed object id
func (s *IdentityService) NewExternalIdentity(ctx context.Context, ID string, identity *IdentityOptions) (*Identity, *Response, error) {
	u := fmt.Sprintf("identity/globalIds/%s/externalIds", ID)

	req, err := s.client.NewRequest("POST", u, "", identity)
	if err != nil {
		return nil, nil, err
	}

	data := new(Identity)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}
