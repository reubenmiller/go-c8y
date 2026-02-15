package model

// ServiceUser has the service user credentials for a given application subscription
type ServiceUser struct {
	Username string `json:"name"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
}

// Binary ManagedObject representation
type Microservice struct {
	ID                string            `json:"id,omitempty"`
	Key               string            `json:"key,omitempty"`
	Name              string            `json:"name,omitempty"`
	Type              string            `json:"type,omitempty"`
	Availability      string            `json:"availability,omitempty"`
	Self              string            `json:"self,omitempty"`
	ContextPath       string            `json:"contextPath,omitempty"`
	ExternalURL       string            `json:"externalUrl,omitempty"`
	ResourcesURL      string            `json:"resourcesUrl,omitempty"`
	ResourcesUsername string            `json:"resourcesUsername,omitempty"`
	ResourcesPassword string            `json:"resourcesPassword,omitempty"`
	Owner             *ApplicationOwner `json:"owner,omitempty"`

	// Microservice roles
	RequiredRoles []string `json:"requiredRoles,omitempty"`
	Roles         []string `json:"roles,omitempty"`

	Manifest *MicroserviceManifest `json:"manifest,omitempty"`
}

// MicroserviceCollection contains information about a list of microservices
type MicroserviceCollection struct {
	*BaseResponse

	Microservices []Microservice `json:"applications"`
}

type MicroserviceManifest struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	RequiredRoles []string `json:"requiredRoles"`
	Roles         []string `json:"roles,omitempty"`
	APIVersion    string   `json:"apiVersion,omitempty"`
}

type MicroserviceReference struct {
	Self        string        `json:"self,omitempty"`
	Application *Microservice `json:"application,omitempty"`
}

// MicroserviceUser containers the credentials to access a given tenant
type MicroserviceUser struct {
	Username string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
	Tenant   string `json:"tenant,omitempty"`
}

type MicroserviceUserCollection struct {
	Users []MicroserviceUser `json:"users"`
}

type MicroserviceSetting struct {
	Key              string `json:"key,omitempty"`
	DefaultValue     string `json:"defaultValue,omitempty"`
	Editable         bool   `json:"editable,omitempty"`
	InheritFromOwner bool   `json:"inheritFromOwner,omitempty"`
	ValueSchema      struct {
		Type string `json:"type,omitempty"`
	} `json:"valueSchema,omitempty"`
}
