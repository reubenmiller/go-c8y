package model

// Identity representation
type Identity struct {
	ExternalID    string                         `json:"externalId"`
	Type          string                         `json:"type"`
	Self          string                         `json:"self"`
	ManagedObject IdentityManagedObjectReference `json:"managedObject"`
}

type IdentityManagedObjectReference struct {
	ID   string `json:"id,omitempty"`
	Self string `json:"self,omitempty"`
}

// IdentityCollection collection of external identities
type IdentityCollection struct {
	*BaseResponse

	Identities []Identity `json:"externalIds,omitempty"`
}
